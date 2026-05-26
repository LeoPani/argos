package inpipatents

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
)

// Config holds pipeline tuning knobs.
type Config struct {
	Concurrency int    // parallel classify+persist workers
	DownloadDir string // local dir for downloaded ZIPs/XMLs
}

// Pipeline orchestrates download → parse → classify → persist.
type Pipeline struct {
	downloader *Downloader
	parser     *Parser
	patentSvc  *service.PatentService
	cfg        Config
	log        *slog.Logger
}

// NewPipeline wires all components together.
func NewPipeline(patentSvc *service.PatentService, log *slog.Logger, cfg Config) *Pipeline {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = "/tmp/argos/rpi"
	}
	return &Pipeline{
		downloader: NewDownloader(cfg.DownloadDir, log),
		parser:     NewParser(log),
		patentSvc:  patentSvc,
		cfg:        cfg,
		log:        log,
	}
}

// Stats accumulates pipeline run metrics.
type Stats struct {
	mu        sync.Mutex
	Parsed    int
	Ingested  int
	Duplicate int
	Failed    int
}

func (s *Stats) incParsed()    { s.mu.Lock(); s.Parsed++; s.mu.Unlock() }
func (s *Stats) incIngested()  { s.mu.Lock(); s.Ingested++; s.mu.Unlock() }
func (s *Stats) incDuplicate() { s.mu.Lock(); s.Duplicate++; s.mu.Unlock() }
func (s *Stats) incFailed()    { s.mu.Lock(); s.Failed++; s.mu.Unlock() }

// RunRPIIssue processes a single RPI issue end-to-end:
//  1. Download XML from INPI
//  2. Stream-parse into Patent records
//  3. Classify + persist via PatentService (with concurrency)
func (p *Pipeline) RunRPIIssue(ctx context.Context, rpiIssue string) (*Stats, error) {
	p.log.Info("pipeline: starting RPI issue", "rpi", rpiIssue)

	result, err := p.downloader.Download(ctx, rpiIssue)
	if err != nil {
		return nil, fmt.Errorf("download RPI %s: %w", rpiIssue, err)
	}
	p.log.Info("pipeline: xml ready", "path", result.LocalPath, "bytes", result.SizeBytes)

	stats := &Stats{}
	jobs := make(chan *domain.Patent, p.cfg.Concurrency*2)

	// Worker goroutines: classify + persist.
	var wg sync.WaitGroup
	for i := 0; i < p.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for patent := range jobs {
				if ctx.Err() != nil {
					return
				}
				_, err := p.patentSvc.Ingest(ctx, patent)
				if err != nil {
					if errors.Is(err, domain.ErrDuplicate) {
						stats.incDuplicate()
					} else {
						stats.incFailed()
						p.log.Warn("pipeline: ingest failed",
							"number", patent.ApplicationNumber, "err", err)
					}
					continue
				}
				stats.incIngested()
			}
		}()
	}

	// Parser feeds the workers.
	parseErr := p.parser.Parse(result.LocalPath, rpiIssue, func(patent *domain.Patent) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		stats.incParsed()
		jobs <- patent
		return nil
	})
	close(jobs)
	wg.Wait()

	if parseErr != nil {
		return stats, fmt.Errorf("parse RPI %s: %w", rpiIssue, parseErr)
	}

	p.log.Info("pipeline: RPI issue done",
		"rpi", rpiIssue,
		"parsed", stats.Parsed,
		"ingested", stats.Ingested,
		"duplicate", stats.Duplicate,
		"failed", stats.Failed,
	)
	return stats, nil
}
