package fetcher

import (
	"TgNewsPet/model"
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"go.tomakado.io/containers/set"
)

type ArticleStorage interface {
	Store(ctx context.Context, article model.Article) error
}

type SourceProvider interface {
	Source(ctx context.Context) ([]model.Source, error)
}

type Source interface {
	ID() int64
	Name() string
	Fetch(ctx context.Context) ([]model.Item, error)
}

type Fetcher struct {
	articles ArticleStorage
	sources  SourceProvider

	fetcherInterval time.Duration
	filterKeywords  []string
}

func New (
	articles ArticleStorage,
	sources SourceProvider,
	fetcherInterval time.Duration,
	filterKeywords []string
	) *Fetcher {
	return &Fetcher{
		articles:        articles,
		sources:         sources,
		fetcherInterval: fetcherInterval,
		filterKeywords:  filterKeywords,
	}
}

func (f *Fetcher) Stert (ctx context.Context) error {
	ticker := time.NewTicker(f.fetcherInterval)
	defer ticker.Stop()

	if err := f.Fetch(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.Fetch(ctx); err != nil {
				return err
			}
		}
	}
}

func (f *Fetcher) Fetch(ctx context.Context) error {
	sources, err := f.sources.Source(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, source := range sources {
		wg.Add(1)

		rssSource := source.NewRSSSourceFromModel(src)

		go func(source Source) {
			defer wg.Done()

			items, err := source.Fetch(ctx)
			if err != nil {
				log.Printf("[ERROR] failed to fetch items from source %q: %v", source.Name(), err)
				return
			}

			if err := f.processItems(ctx, source, items); err != nil {
				log.Printf("[ERROR] failed to process items from source %q: %v", source.Name(), err)
				return
			}
		}(rssSource)
	}

	wg.Wait()
	return nil
}

func (f *Fetcher) processItems(ctx context.Context, source Source, items []model.Item) error {
	for _, item := range items {
		item.Date = item.Date.UTC()

		if f.itemShouldBeSkipped(item){
			continue
		}

		if err := f.articles.Store(ctx, model.Article{
			SourceID: source.ID(),
			Title: 		 item.Title,
			Link:		 item.Link,
			Summary: 	 item.Summary,
			PublishedAt: item.Date,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (f *Fetcher) itemShouldBeSkipped(item model.Item) bool {
	categoriesSet:= set.New(item.Categories...)

	for _, keyword := range f.filterKeywords {
		titleContainsKeyword := strings.Contains(strings.ToLower(item.Title), keyword)

		if categoriesSet.Contains(keyword) || titleContainsKeyword {
			return true
		}
	}
	return false
}