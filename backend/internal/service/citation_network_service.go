// Package service — CitationNetworkService produz o grafo de citações
// (forward + backward) ao redor de uma patente UFOP.
//
// Formato compatível com bibliotecas force-directed (react-force-graph,
// D3): { nodes: [{id, label, group}], links: [{source, target}] }.
//
// Validação acadêmica: análise de redes de citação é padrão na patentometria
// desde Narin (1994) e formalizado por Hall, Jaffe & Trajtenberg (2001).
package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// CitationNode é um vértice do grafo.
type CitationNode struct {
	ID     string `json:"id"`     // unique key
	Label  string `json:"label"`  // displayed text
	Group  string `json:"group"`  // "self" | "forward" | "backward"
	Year   int    `json:"year,omitempty"`
	IPC    string `json:"ipc,omitempty"`
}

// CitationLink é uma aresta entre dois nós.
type CitationLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"` // "forward" | "backward"
}

// CitationNetwork é o grafo completo.
type CitationNetwork struct {
	Nodes  []CitationNode `json:"nodes"`
	Links  []CitationLink `json:"links"`
	Center string         `json:"center_node_id"`
	Stats  NetworkStats   `json:"stats"`
}

// NetworkStats sumariza o grafo.
type NetworkStats struct {
	NodeCount     int     `json:"node_count"`
	ForwardCount  int     `json:"forward_count"`
	BackwardCount int     `json:"backward_count"`
	AvgYear       float64 `json:"avg_year"`
}

// CitationNetworkService.
type CitationNetworkService struct{ db *sql.DB }

func NewCitationNetworkService(db *sql.DB) *CitationNetworkService {
	return &CitationNetworkService{db: db}
}

// Build constrói o grafo a partir das citações persistidas em patent_citations.
func (s *CitationNetworkService) Build(ctx context.Context, patentID int64) (*CitationNetwork, error) {
	// Center node: a patente principal
	var (
		appNum string
		title  string
		ipcCat sql.NullInt64
		year   sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT application_number, title,
		       COALESCE(ipc_category, -1),
		       EXTRACT(YEAR FROM filing_date)::INT
		FROM patents WHERE id = $1`, patentID).
		Scan(&appNum, &title, &ipcCat, &year)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("patent id=%d not found", patentID)
	}
	if err != nil {
		return nil, err
	}

	centerID := fmt.Sprintf("p%d", patentID)
	network := &CitationNetwork{
		Center: centerID,
		Nodes: []CitationNode{{
			ID: centerID, Label: appNum, Group: "self",
		}},
		Links: []CitationLink{},
	}
	if year.Valid {
		network.Nodes[0].Year = int(year.Int64)
	}
	if ipcCat.Valid && ipcCat.Int64 >= 0 && ipcCat.Int64 < 8 {
		network.Nodes[0].IPC = ipcLetters[ipcCat.Int64]
	}

	// Pull citations
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, citation_kind, cited_app_number, cited_title, cited_year, cited_ipc_codes
		FROM patent_citations
		WHERE source_patent_id = $1
		ORDER BY citation_kind, cited_year DESC`, patentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var yearSum, yearN int
	for rows.Next() {
		var (
			cid       int64
			kind      string
			appNum    sql.NullString
			title     sql.NullString
			cYear     sql.NullInt64
			ipcCodes  []string
		)
		if err := rows.Scan(&cid, &kind, &appNum, &title, &cYear, pq.Array(&ipcCodes)); err != nil {
			return nil, err
		}

		nodeID := fmt.Sprintf("c%d", cid)
		label := appNum.String
		if label == "" && title.Valid {
			label = title.String
			if len(label) > 30 {
				label = label[:30] + "…"
			}
		}
		node := CitationNode{
			ID: nodeID, Label: label, Group: kind,
		}
		if cYear.Valid {
			node.Year = int(cYear.Int64)
			yearSum += node.Year
			yearN++
		}
		if len(ipcCodes) > 0 && len(ipcCodes[0]) > 0 {
			node.IPC = string(ipcCodes[0][0]) // section letter
		}
		network.Nodes = append(network.Nodes, node)

		// Link direction:
		//   forward = SOMEONE cited us         (other → self)
		//   backward = WE cited someone        (self → other)
		if kind == "forward" {
			network.Links = append(network.Links, CitationLink{
				Source: nodeID, Target: centerID, Kind: kind,
			})
			network.Stats.ForwardCount++
		} else {
			network.Links = append(network.Links, CitationLink{
				Source: centerID, Target: nodeID, Kind: kind,
			})
			network.Stats.BackwardCount++
		}
	}

	network.Stats.NodeCount = len(network.Nodes)
	if yearN > 0 {
		network.Stats.AvgYear = float64(yearSum) / float64(yearN)
	}

	return network, nil
}
