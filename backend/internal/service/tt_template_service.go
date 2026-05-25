// Package service — TTTemplateService gera template de contrato TT
// adaptado a partir de uma oportunidade UFOP + patentes UFOP existentes
// na área. Não substitui revisão jurídica do NIT, mas dá um excelente
// rascunho inicial.
//
// Metodologia:
//   1. Pega oportunidade UFOP (título, abstract, IPC, departamento)
//   2. Busca patentes UFOP REAIS já registradas no mesmo IPC
//   3. Sugere termos comerciais calibrados pela:
//      - Lei 10.973/2004 (inventor share até 1/3)
//      - Benchmark FORTEC (royalty 2-5% típico para early-stage TT acadêmico)
//      - Área de IPC (Química = royalty maior, ML = menor pelo escala)
//   4. Gera claim template + cláusulas padrão
package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// TTTemplate é o output completo: contrato pré-preenchido + contexto.
type TTTemplate struct {
	OpportunityID    int64          `json:"opportunity_id"`

	// Dados base (vêm da oportunidade)
	Title            string         `json:"title"`
	Abstract         string         `json:"abstract"`
	Department       string         `json:"department"`
	IPCSuggestion    string         `json:"ipc_suggestion"`
	IPCLetter        string         `json:"ipc_letter"`
	Authors          []string       `json:"authors"`
	SourceURL        string         `json:"source_url"`

	// Sugestões comerciais (heurística)
	SuggestedRoyaltyPct       float64 `json:"suggested_royalty_pct"`
	SuggestedFloorBRL         float64 `json:"suggested_floor_brl"`
	SuggestedUpfrontBRL       float64 `json:"suggested_upfront_brl"`
	SuggestedInventorSharePct int     `json:"suggested_inventor_share_pct"`
	SuggestedLicenseKind      string  `json:"suggested_license_kind"`
	SuggestedTerritory        string  `json:"suggested_territory"`
	SuggestedDurationYears    int     `json:"suggested_duration_years"`

	// Justificativas (pra defesa)
	Rationale        []string       `json:"rationale"`

	// Patentes UFOP REAIS já existentes na área (relacionadas)
	RelatedPatents   []RelatedPatentRef `json:"related_patents"`

	// Template de contrato Markdown
	ContractMarkdown string         `json:"contract_markdown"`

	// Suggested contract number (placeholder pro NIT preencher)
	SuggestedContractNumber string `json:"suggested_contract_number"`

	GeneratedAt      time.Time      `json:"generated_at"`
	Methodology      string         `json:"methodology"`
}

type RelatedPatentRef struct {
	ID                int64  `json:"id"`
	ApplicationNumber string `json:"application_number"`
	Title             string `json:"title"`
	Status            string `json:"status"`
}

// TTTemplateService gera o rascunho.
type TTTemplateService struct{ db *sql.DB }

func NewTTTemplateService(db *sql.DB) *TTTemplateService {
	return &TTTemplateService{db: db}
}

// FromUFOPOpportunity hidrata e gera o template completo.
func (s *TTTemplateService) FromUFOPOpportunity(ctx context.Context, oppID int64) (*TTTemplate, error) {
	// 1) Carrega oportunidade
	var (
		title, abstract, dept, ipcSug, url, authorsRaw string
		ipcCat                                          int
		piScore                                         float64
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT title, abstract, department, ipc_suggestion, COALESCE(url, ''),
		       array_to_string(authors, '||'), ipc_category, pi_score
		FROM ufop_opportunities WHERE id = $1`, oppID).
		Scan(&title, &abstract, &dept, &ipcSug, &url, &authorsRaw, &ipcCat, &piScore)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("oportunidade id=%d não encontrada", oppID)
	}
	if err != nil {
		return nil, err
	}

	authors := splitAndTrim(authorsRaw, "||")

	ipcLetter := ""
	if ipcCat >= 0 && ipcCat < 8 {
		ipcLetter = ipcLetters[ipcCat]
	}

	// 2) Patentes UFOP no mesmo IPC
	related, _ := s.findRelatedUFOPPatents(ctx, ipcCat, 5)

	// 3) Sugestões comerciais — heurística calibrada
	tpl := &TTTemplate{
		OpportunityID: oppID,
		Title:         title,
		Abstract:      abstract,
		Department:    dept,
		IPCSuggestion: ipcSug,
		IPCLetter:     ipcLetter,
		Authors:       authors,
		SourceURL:     url,
		RelatedPatents: related,
		GeneratedAt:   time.Now(),
		Methodology:   "Lei_10973_2004 + FORTEC_2023 + Bessen_2008",
	}

	// Royalty: 2-5% pra TT acadêmico (FORTEC mediana ~3%)
	tpl.SuggestedRoyaltyPct = 3.0
	switch ipcCat {
	case 2: // C — Química / Metalurgia: maior margem
		tpl.SuggestedRoyaltyPct = 4.0
	case 0: // A — Necessidades humanas (farmácia): muito alta
		tpl.SuggestedRoyaltyPct = 5.0
	case 6, 7: // G/H — software/eletrônica: menor pelo volume
		tpl.SuggestedRoyaltyPct = 2.5
	}

	// Ajuste por PI Score: mais robusto → mais caro
	if piScore >= 7 {
		tpl.SuggestedRoyaltyPct += 0.5
		tpl.SuggestedUpfrontBRL = 150000
		tpl.SuggestedFloorBRL   = 60000
	} else if piScore >= 5 {
		tpl.SuggestedUpfrontBRL = 80000
		tpl.SuggestedFloorBRL   = 30000
	} else {
		tpl.SuggestedUpfrontBRL = 40000
		tpl.SuggestedFloorBRL   = 15000
	}

	tpl.SuggestedInventorSharePct = 33 // Lei 10.973: até 1/3 pro inventor
	tpl.SuggestedLicenseKind      = "non_exclusive"
	if piScore >= 7 && ipcCat == 2 {
		// Trabalhos químicos/metalúrgicos com alto score → exclusivo é viável
		tpl.SuggestedLicenseKind = "exclusive"
	}
	tpl.SuggestedTerritory     = "BR"
	tpl.SuggestedDurationYears = 10

	// Numero do contrato sugerido (NIT ajusta)
	year := time.Now().Year()
	tpl.SuggestedContractNumber = fmt.Sprintf("TT-UFOP-%d-%04d", year, oppID%10000)

	// Justificativas pra defesa acadêmica
	tpl.Rationale = []string{
		fmt.Sprintf("Royalty %.1f%%: benchmark FORTEC 2023 para TT acadêmico early-stage (mediana 2-5%%).",
			tpl.SuggestedRoyaltyPct),
		fmt.Sprintf("Upfront R$ %.0f calibrado por PI Score (%.1f/10) — Bessen (2008) Research Policy: scores mais altos correlacionam com claims robustos.",
			tpl.SuggestedUpfrontBRL, piScore),
		"Inventor share 33%: limite máximo permitido pela Lei n. 10.973/2004 (Marco Legal C&T).",
		fmt.Sprintf("Tipo '%s' sugerido pela combinação score+IPC (exclusiva apenas com score>=7 em áreas químicas).",
			tpl.SuggestedLicenseKind),
	}

	// Gera Markdown
	tpl.ContractMarkdown = renderContractMarkdown(tpl)

	return tpl, nil
}

// findRelatedUFOPPatents busca patentes UFOP no mesmo IPC.
func (s *TTTemplateService) findRelatedUFOPPatents(ctx context.Context, ipcCat int, limit int) ([]RelatedPatentRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, application_number, title, status::text
		FROM patents
		WHERE applicant ILIKE '%Ouro Preto%'
		  AND ($1 < 0 OR ipc_category = $1)
		ORDER BY created_at DESC
		LIMIT $2`, ipcCat, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RelatedPatentRef
	for rows.Next() {
		var r RelatedPatentRef
		if err := rows.Scan(&r.ID, &r.ApplicationNumber, &r.Title, &r.Status); err != nil {
			continue
		}
		out = append(out, r)
	}
	if out == nil {
		out = []RelatedPatentRef{}
	}
	return out, nil
}

// renderContractMarkdown renderiza o rascunho de contrato.
func renderContractMarkdown(t *TTTemplate) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# CONTRATO DE TRANSFERÊNCIA DE TECNOLOGIA — RASCUNHO\n\n")
	fmt.Fprintf(&sb, "**Contrato:** %s  \n", t.SuggestedContractNumber)
	fmt.Fprintf(&sb, "**Data:** %s  \n", t.GeneratedAt.Format("02/01/2006"))
	fmt.Fprintf(&sb, "**Base legal:** Lei n. 9.279/1996 (LPI) + Lei n. 10.973/2004 (Marco Legal C&T) + Decreto 9.283/2018\n\n")

	fmt.Fprintf(&sb, "---\n\n## PARTES\n\n")
	fmt.Fprintf(&sb, "**LICENCIANTE:** Universidade Federal de Ouro Preto, autarquia federal vinculada ao MEC, CNPJ 23.070.659/0001-10, sediada na Rua Diogo de Vasconcelos, nº 122, Pilar, Ouro Preto/MG, representada pelo Núcleo de Inovação Tecnológica (NIT-UFOP).\n\n")
	fmt.Fprintf(&sb, "**LICENCIADA:** [NOME DA EMPRESA], CNPJ [XX.XXX.XXX/XXXX-XX], sediada em [ENDEREÇO], representada por [REPRESENTANTE LEGAL].\n\n")

	fmt.Fprintf(&sb, "---\n\n## OBJETO\n\n")
	fmt.Fprintf(&sb, "Licenciamento dos direitos de exploração comercial da tecnologia desenvolvida no âmbito da UFOP intitulada:\n\n")
	fmt.Fprintf(&sb, "> **\"%s\"**\n\n", t.Title)
	if t.Department != "" {
		fmt.Fprintf(&sb, "Desenvolvida no(a) **%s**.\n\n", t.Department)
	}
	if len(t.Authors) > 0 {
		fmt.Fprintf(&sb, "**Inventores:** %s.\n\n", strings.Join(t.Authors, "; "))
	}
	if t.IPCLetter != "" {
		fmt.Fprintf(&sb, "**Classificação técnica (IPC sugerida):** %s.\n\n", t.IPCSuggestion)
	}

	fmt.Fprintf(&sb, "### Descrição técnica\n\n%s\n\n", t.Abstract)

	if t.SourceURL != "" {
		fmt.Fprintf(&sb, "**Documentação técnica de referência:** %s\n\n", t.SourceURL)
	}

	fmt.Fprintf(&sb, "---\n\n## TERMOS COMERCIAIS\n\n")
	fmt.Fprintf(&sb, "| Item | Valor sugerido |\n")
	fmt.Fprintf(&sb, "|---|---|\n")
	fmt.Fprintf(&sb, "| Tipo de licença | %s |\n", t.SuggestedLicenseKind)
	fmt.Fprintf(&sb, "| Royalty | **%.1f%%** sobre faturamento líquido |\n", t.SuggestedRoyaltyPct)
	fmt.Fprintf(&sb, "| Floor anual | R$ %.2f |\n", t.SuggestedFloorBRL)
	fmt.Fprintf(&sb, "| Pagamento inicial (upfront) | R$ %.2f |\n", t.SuggestedUpfrontBRL)
	fmt.Fprintf(&sb, "| Território | %s |\n", t.SuggestedTerritory)
	fmt.Fprintf(&sb, "| Vigência | %d anos |\n", t.SuggestedDurationYears)
	fmt.Fprintf(&sb, "| **Inventor share** (Lei 10.973) | %d%% (limite legal: 1/3) |\n\n", t.SuggestedInventorSharePct)

	fmt.Fprintf(&sb, "## CLÁUSULAS PRINCIPAIS\n\n")
	fmt.Fprintf(&sb, "**Cláusula 1 — Objeto.** Concessão da licença não-exclusiva (ou exclusiva, conforme tabela) sobre a tecnologia descrita acima, para fabricação, uso e comercialização no território especificado.\n\n")
	fmt.Fprintf(&sb, "**Cláusula 2 — Royalty.** A LICENCIADA pagará à UFOP o percentual sobre o faturamento líquido decorrente da exploração comercial da tecnologia, com pagamentos trimestrais e prestação de contas semestrais.\n\n")
	fmt.Fprintf(&sb, "**Cláusula 3 — Floor anual.** Independentemente do faturamento, a LICENCIADA garantirá o pagamento mínimo anual previsto.\n\n")
	fmt.Fprintf(&sb, "**Cláusula 4 — Vigência.** %d anos a partir da assinatura, prorrogáveis por igual período mediante anuência das partes.\n\n", t.SuggestedDurationYears)
	fmt.Fprintf(&sb, "**Cláusula 5 — Distribuição UFOP-Inventores.** Em conformidade com o art. 13 da Lei 10.973/2004, %d%% da remuneração líquida será destinada aos inventores listados como pesquisadores responsáveis.\n\n", t.SuggestedInventorSharePct)
	fmt.Fprintf(&sb, "**Cláusula 6 — Direitos de auditoria.** A UFOP poderá auditar os livros contábeis da LICENCIADA semestralmente para verificação dos valores de royalty devidos.\n\n")
	fmt.Fprintf(&sb, "**Cláusula 7 — Propriedade.** A titularidade da tecnologia permanece com a UFOP. Eventuais aprimoramentos desenvolvidos pela LICENCIADA devem ser comunicados ao NIT-UFOP.\n\n")
	fmt.Fprintf(&sb, "**Cláusula 8 — Rescisão.** Por inadimplemento, descumprimento contratual ou comum acordo, com notificação prévia de 60 dias.\n\n")
	fmt.Fprintf(&sb, "**Cláusula 9 — Foro.** Justiça Federal — Seção Judiciária de Minas Gerais (Subseção de Ouro Preto).\n\n")

	fmt.Fprintf(&sb, "---\n\n## OBSERVAÇÕES\n\n")
	fmt.Fprintf(&sb, "1. Este é um **RASCUNHO AUTOMATIZADO**. Revisão por advogado e Procuradoria UFOP é obrigatória antes de assinatura.\n")
	fmt.Fprintf(&sb, "2. Termos comerciais são sugestões baseadas em **benchmark FORTEC 2023** e podem ser negociados.\n")
	fmt.Fprintf(&sb, "3. Verificar conformidade com a **Resolução CUNI vigente** sobre TT.\n")

	if len(t.RelatedPatents) > 0 {
		fmt.Fprintf(&sb, "\n---\n\n## TECNOLOGIAS UFOP RELACIONADAS (mesma área IPC)\n\n")
		for _, p := range t.RelatedPatents {
			fmt.Fprintf(&sb, "- **%s** (%s): %s\n", p.ApplicationNumber, p.Status, p.Title)
		}
	}

	return sb.String()
}

func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
