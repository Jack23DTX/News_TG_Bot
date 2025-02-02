package summary

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"strings"
	"sync"
	"time"
)

type OpenAISummarizer struct {
	client  *openai.Client
	prompt  string
	model   string
	enabled bool
	mu      sync.Mutex
}

func NewOpenAISummarizer(apiKey, model, prompt string) *OpenAISummarizer {
	s := &OpenAISummarizer{
		client: openai.NewClient(apiKey),
		prompt: prompt,
		model:  model,
	}

	log.Printf("openai summarizer enabled: %v", apiKey != "")
	if apiKey != "" {
		s.enabled = true
	}

	return s
}

func (s *OpenAISummarizer) Summarize(text string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.enabled {
		return "", nil
	}

	request := openai.ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf("%s%s", text, s.prompt),
			},
		},
		MaxTokens:   256,
		Temperature: 0.7,
		TopP:        1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", err
	}

	rawSummary := strings.TrimSpace(resp.Choices[0].Message.Content)

	if strings.HasSuffix(rawSummary, ".") {
		return rawSummary, nil
	}

	sentence := strings.Split(rawSummary, ".")

	return strings.Join(sentence[:len(sentence)-1], ".") + ".", nil
}
