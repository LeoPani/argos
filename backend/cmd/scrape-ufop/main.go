package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/LeoPani/argos/backend/internal/worker/ufop"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	scraper := ufop.NewPortalScraper(log)
	news, err := scraper.ScrapeNews(ctx)
	if err != nil {
		log.Error("scrape failed", "err", err)
		return
	}

	log.Info("scraped news", "count", len(news))
	for i, n := range news {
		if i >= 10 {
			break
		}
		slog.Info("  match", "kw", n.Keywords, "title", n.Title)
	}
}
