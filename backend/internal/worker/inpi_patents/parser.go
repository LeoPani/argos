package inpipatents

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// ─── INPI XML schema (simplified BRPI 2.0 subset) ────────────────────────────

type rpiXML struct {
	XMLName  xml.Name    `xml:"br-patent-document"`
	BRNumber string      `xml:"br-number,attr"`
	FilingDate string    `xml:"br-filing-date,attr"`
	Invention inventionXML `xml:"invention-title"`
	Abstract  abstractXML  `xml:"abstract"`
	Parties   partiesXML   `xml:"parties"`
	IPCData   []ipcXML     `xml:"classifications-ipcr>classification-ipcr"`
}

type inventionXML struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

type abstractXML struct {
	Text string `xml:",chardata"`
}

type partiesXML struct {
	Applicants []applicantXML `xml:"applicants>applicant"`
	Inventors  []inventorXML  `xml:"inventors>inventor"`
}

type applicantXML struct {
	Name string `xml:"addressbook>name"`
}

type inventorXML struct {
	Name string `xml:"addressbook>name"`
}

type ipcXML struct {
	Section   string `xml:"section"`
	Class     string `xml:"class"`
	Subclass  string `xml:"subclass"`
}

// ─── Parser ──────────────────────────────────────────────────────────────────

// Parser streams-parses an INPI XML file and emits domain.Patent records.
type Parser struct {
	log *slog.Logger
}

// NewParser creates a Parser.
func NewParser(log *slog.Logger) *Parser { return &Parser{log: log} }

// Parse opens the XML file at path and calls fn for each patent.
// fn returning an error stops parsing and propagates the error.
func (p *Parser) Parse(path string, rpiIssue string, fn func(*domain.Patent) error) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open xml %s: %w", path, err)
	}
	defer f.Close()
	return p.parseReader(f, rpiIssue, fn)
}

func (p *Parser) parseReader(r io.Reader, rpiIssue string, fn func(*domain.Patent) error) error {
	dec := xml.NewDecoder(r)
	dec.Strict = false

	count := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("xml token: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "br-patent-document" {
			continue
		}

		var raw rpiXML
		if err := dec.DecodeElement(&raw, &se); err != nil {
			p.log.Warn("inpi parser: decode element error, skipping", "err", err)
			continue
		}

		patent, err := mapRPIToPatent(&raw, rpiIssue)
		if err != nil {
			p.log.Warn("inpi parser: map error, skipping", "number", raw.BRNumber, "err", err)
			continue
		}

		if err := fn(patent); err != nil {
			return err
		}
		count++
	}
	p.log.Info("inpi parser: finished", "count", count, "rpi", rpiIssue)
	return nil
}

func mapRPIToPatent(raw *rpiXML, rpiIssue string) (*domain.Patent, error) {
	if raw.BRNumber == "" {
		return nil, fmt.Errorf("missing br-number")
	}

	var applicant string
	if len(raw.Parties.Applicants) > 0 {
		applicant = strings.TrimSpace(raw.Parties.Applicants[0].Name)
	}

	inventors := make([]string, 0, len(raw.Parties.Inventors))
	for _, inv := range raw.Parties.Inventors {
		if n := strings.TrimSpace(inv.Name); n != "" {
			inventors = append(inventors, n)
		}
	}

	var ipcCode string
	if len(raw.IPCData) > 0 {
		ipc := raw.IPCData[0]
		ipcCode = strings.TrimSpace(ipc.Section + ipc.Class + ipc.Subclass)
	}

	var filingDate *time.Time
	if raw.FilingDate != "" {
		for _, layout := range []string{"2006-01-02", "20060102", "02/01/2006"} {
			if t, err := time.Parse(layout, strings.TrimSpace(raw.FilingDate)); err == nil {
				filingDate = &t
				break
			}
		}
	}

	title := strings.TrimSpace(raw.Invention.Value)
	if title == "" {
		return nil, fmt.Errorf("missing title for %s", raw.BRNumber)
	}

	return &domain.Patent{
		ApplicationNumber: strings.TrimSpace(raw.BRNumber),
		Title:             title,
		Abstract:          strings.TrimSpace(raw.Abstract.Text),
		Applicant:         applicant,
		Inventors:         inventors,
		FilingDate:        filingDate,
		IPCCode:           ipcCode,
		RPIIssue:          rpiIssue,
		Status:            domain.PatentStatusPending,
	}, nil
}
