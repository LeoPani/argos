// Package inpipatents implements the INPI patent ingestion pipeline.
//
// INPI publishes weekly XML data dumps at:
//   https://www.gov.br/inpi/pt-br/servicos/patentes/xml-de-dados-abertos
//
// Each ZIP contains one XML file per RPI issue (Revista da Propriedade
// Industrial). The XML is gzipped and follows the BRPI schema.
package inpipatents

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// inpiBaseURL is the root URL for INPI open data XML downloads.
	inpiBaseURL = "https://www.gov.br/inpi/pt-br/servicos/patentes/xml-de-dados-abertos"
	userAgent   = "Argos-IP-Intelligence/1.0 (+https://github.com/LeoPani/argos)"
)

// Downloader fetches INPI RPI XML ZIPs and stores them locally.
type Downloader struct {
	downloadDir string
	client      *http.Client
	log         *slog.Logger
}

// NewDownloader creates a Downloader that saves files under downloadDir.
func NewDownloader(downloadDir string, log *slog.Logger) *Downloader {
	return &Downloader{
		downloadDir: downloadDir,
		client: &http.Client{
			Timeout: 10 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		log: log,
	}
}

// DownloadResult is returned by Download.
type DownloadResult struct {
	LocalPath string
	RPIIssue  string
	SizeBytes int64
}

// Download fetches the XML ZIP for a specific RPI issue number and saves it.
// Returns the local path to the unpacked XML file.
func (d *Downloader) Download(ctx context.Context, rpiIssue string) (*DownloadResult, error) {
	// Ensure download directory exists.
	if err := os.MkdirAll(d.downloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", d.downloadDir, err)
	}

	zipPath := filepath.Join(d.downloadDir, fmt.Sprintf("rpi_%s.zip", rpiIssue))
	xmlPath := filepath.Join(d.downloadDir, fmt.Sprintf("rpi_%s.xml", rpiIssue))

	// Skip download if XML already exists (idempotent).
	if info, err := os.Stat(xmlPath); err == nil {
		d.log.Info("inpi: xml already downloaded, skipping", "rpi", rpiIssue, "path", xmlPath)
		return &DownloadResult{LocalPath: xmlPath, RPIIssue: rpiIssue, SizeBytes: info.Size()}, nil
	}

	url := fmt.Sprintf("%s/rpi-%s.zip", inpiBaseURL, rpiIssue)
	d.log.Info("inpi: downloading RPI", "rpi", rpiIssue, "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download RPI %s: %w", rpiIssue, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download RPI %s: HTTP %d", rpiIssue, resp.StatusCode)
	}

	// Save ZIP.
	f, err := os.Create(zipPath)
	if err != nil {
		return nil, fmt.Errorf("create zip file: %w", err)
	}
	n, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(zipPath)
		return nil, fmt.Errorf("write zip: %w", err)
	}
	d.log.Info("inpi: downloaded zip", "rpi", rpiIssue, "bytes", n)

	// Unzip — find the XML inside.
	xmlSize, err := d.extractXML(zipPath, xmlPath)
	if err != nil {
		return nil, fmt.Errorf("extract xml from zip: %w", err)
	}
	os.Remove(zipPath) // clean up zip after extraction

	return &DownloadResult{LocalPath: xmlPath, RPIIssue: rpiIssue, SizeBytes: xmlSize}, nil
}

func (d *Downloader) extractXML(zipPath, destPath string) (int64, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Ext(f.Name) != ".xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return 0, fmt.Errorf("open file in zip: %w", err)
		}
		out, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return 0, fmt.Errorf("create xml file: %w", err)
		}
		n, err := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return 0, fmt.Errorf("extract xml: %w", err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("no XML file found inside %s", zipPath)
}
