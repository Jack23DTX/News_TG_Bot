package notifier

import (
	"TgNewsPet/botkit/markup"
	"TgNewsPet/model"
	"context"
	"fmt"
	"github.com/go-shiori/go-readability"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type ArticleProvider interface {
	AllNotPosted(ctx context.Context, since time.Time, limit int64) ([]model.Article, error)
	MarkPosted(ctx context.Context, id int64) error
}

type Summarizer interface {
	summarize(ctx context.Context, text string) (string, error)
}

type Notifier struct {
	articles         ArticleProvider
	summarizer       Summarizer
	bot              *tgbotapi.BotAPI
	sendInterval     time.Duration
	lookupTimeWindow time.Duration
	channelID        int64
}

func New(
	articleProvider ArticleProvider,
	summarizer Summarizer,
	bot *tgbotapi.BotAPI,
	sendInterval time.Duration,
	lookupTimeWindow time.Duration,
	channelID int64,
) *Notifier {
	return &Notifier{
		articles:         articleProvider,
		summarizer:       summarizer,
		bot:              bot,
		sendInterval:     sendInterval,
		lookupTimeWindow: lookupTimeWindow,
		channelID:        channelID,
	}
}

func (n *Notifier) SelectAndSendArticle(ctx context.Context) error {
	tpOneArticle, err := n.articles.AllNotPosted(ctx, time.Now().Add(-n.lookupTimeWindow), 1)
	if err != nil {
		return err
	}

	if len(tpOneArticle) == 0 {
		return nil
	}

	article := tpOneArticle[0]

	summary, err := n.extractSummary(ctx, article)
	if err != nil {
		return err
	}

	if err := n.sendArticle(article, summary); err != nil {
		return err
	}

	return n.articles.MarkPosted(ctx, article.ID)
}

var redundantNewLines = regexp.MustCompile(`\n{3,}`)

func (n *Notifier) extractArticle(ctx context.Context, article model.Article) (string, error) {
	var r io.Reader

	if article.Summary != "" {
		r = strings.NewReader(article.Summary)
	} else {
		resp, err := http.Get(article.Link)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		r = resp.Body
	}

	doc, err := readability.FromReader(r, nil)
	if err != nil {
		return "", err
	}

	summary, err := n.summarizer.summarize(ctx, cleanText(doc.TextContent))
	if err != nil {
		return "", err
	}

	return "\n\n" + summary, nil
}

func cleanText(text string) string {
	return redundantNewLines.ReplaceAllString(text, "\n")
}

func (n *Notifier) sendArticle(article model.Article, summary string) error {
	const msgFormat = "*%s*%s\n\n%s"

	msg := tgbotapi.NewMessage(n.channelID, fmt.Sprintf(
		msgFormat,
		markup.EscapeForMarkdown(article.Title),
		markup.EscapeForMarkdown(summary),
		markup.EscapeForMarkdown(article.Link),
	))
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	_, err := n.bot.Send(msg)
	if err != nil {
		return err
	}
	return nil
}
