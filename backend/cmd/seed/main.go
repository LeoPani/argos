// argos-seed populates the database with realistic synthetic Brazilian
// IP data for demo and development purposes.
//
// Usage:
//
//	make seed
//	# or
//	DATABASE_URL=postgres://... go run ./cmd/seed
//
// Inserts ~100 patents, ~30 trademarks, ~5 disputes and ~20 UFOP
// opportunities, spread across all IPC categories and several years.
// All inserts are idempotent — re-running is safe.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://argos:argos_dev@localhost:5432/argos?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	slog.Info("seed: connected to database")

	rng := rand.New(rand.NewSource(42)) // deterministic

	nPat, err := seedPatents(ctx, db, rng)
	if err != nil {
		return fmt.Errorf("seed patents: %w", err)
	}
	nTm, err := seedTrademarks(ctx, db, rng)
	if err != nil {
		return fmt.Errorf("seed trademarks: %w", err)
	}
	nDis, err := seedDisputes(ctx, db, rng)
	if err != nil {
		return fmt.Errorf("seed disputes: %w", err)
	}
	nUfop, err := seedUFOP(ctx, db, rng)
	if err != nil {
		return fmt.Errorf("seed ufop: %w", err)
	}

	slog.Info("seed: done",
		"patents", nPat, "trademarks", nTm,
		"disputes", nDis, "ufop_opportunities", nUfop,
	)
	return nil
}

// ─── Patents ──────────────────────────────────────────────────────────────────

type patentTemplate struct {
	cat      int    // IPC 0..7
	ipcCode  string // e.g. "C22B"
	titleFmt string // %s gets a noun
	abstract string
}

var patentTemplates = []patentTemplate{
	// Cat 0 — A (Necessidades humanas)
	{0, "A61K", "Composição farmacêutica para tratamento de %s", "Formulação oral de liberação controlada com perfil farmacocinético otimizado."},
	{0, "A23L", "Processo de elaboração de %s com propriedades funcionais", "Método de produção que preserva compostos bioativos e melhora estabilidade."},
	{0, "A61B", "Dispositivo médico para diagnóstico de %s", "Equipamento portátil com sensores e interpretação automatizada de sinais."},

	// Cat 1 — B (Operações / Transportes)
	{1, "B01D", "Membrana de separação para %s", "Membrana polimérica com seletividade elevada e baixo fouling."},
	{1, "B22F", "Processo metalúrgico de pó para %s", "Sinterização assistida que reduz porosidade e melhora propriedades mecânicas."},
	{1, "B05B", "Sistema de pulverização para %s", "Bicos com geometria otimizada que reduzem consumo em até 30%."},

	// Cat 2 — C (Química / Metalurgia)
	{2, "C07D", "Composto heterocíclico aplicável a %s", "Síntese eficiente em um passo com rendimento superior a 85%."},
	{2, "C12N", "Microorganismo recombinante produtor de %s", "Cepa modificada com expressão otimizada de genes-alvo."},
	{2, "C22B", "Processo hidrometalúrgico para extração de %s", "Lixiviação seletiva com baixo consumo de reagentes e impacto ambiental reduzido."},
	{2, "C08L", "Composição polimérica com propriedades aprimoradas para %s", "Blenda termoplástica com módulo elástico 40% superior."},

	// Cat 3 — D (Têxteis / Papel)
	{3, "D04H", "Tecido não-tecido com aplicação em %s", "Fibras orientadas que fornecem resistência e respirabilidade superiores."},
	{3, "D01F", "Fibra sintética para uso em %s", "Polímero modificado com propriedades anti-microbianas duradouras."},

	// Cat 4 — E (Construção civil)
	{4, "E04B", "Sistema construtivo modular para %s", "Painéis pré-fabricados que reduzem tempo de obra em 50%."},
	{4, "E02D", "Estrutura de fundação para %s", "Geometria otimizada para solos de baixa capacidade de suporte."},

	// Cat 5 — F (Engenharia Mecânica)
	{5, "F03D", "Turbina eólica de pequena escala para %s", "Geometria de pás otimizada via CFD para baixa velocidade de vento."},
	{5, "F16C", "Mancal de baixo atrito para %s", "Geometria com revestimento DLC que reduz desgaste e perdas energéticas."},
	{5, "F25B", "Sistema de refrigeração eficiente para %s", "Ciclo termodinâmico modificado com COP 25% superior."},

	// Cat 6 — G (Física / TI)
	{6, "G06F", "Sistema computacional para %s", "Arquitetura distribuída com balanceamento adaptativo de carga."},
	{6, "G06N", "Método de aprendizado de máquina para %s", "Modelo neural otimizado com redução de 60% no custo de inferência."},
	{6, "G01N", "Sensor óptico para detecção de %s", "Princípio plasmônico com limite de detecção em escala ppb."},
	{6, "G06Q", "Plataforma digital para gestão de %s", "Arquitetura microsserviços com rastreabilidade ponta-a-ponta."},

	// Cat 7 — H (Eletricidade / Eletrônica)
	{7, "H01L", "Dispositivo semicondutor para %s", "Estrutura heterogênea que melhora eficiência quântica em 35%."},
	{7, "H02J", "Sistema de gestão de energia em %s", "Controle inteligente que reduz perdas em microrredes isoladas."},
	{7, "H04L", "Protocolo de comunicação para %s", "Esquema criptográfico de baixa latência com perfeita confidencialidade direta."},
	{7, "H01M", "Bateria de íon-lítio para %s", "Eletrodo de cátodo de alta densidade energética e vida útil 2x maior."},
}

var patentNouns = []string{
	"diabetes tipo 2", "câncer de mama", "doenças respiratórias",
	"alimentos funcionais", "bebidas fermentadas", "suplementos nutricionais",
	"águas industriais", "efluentes contaminados", "gases poluentes",
	"minério de ferro", "lítio", "cobre", "ouro de baixo teor", "níquel",
	"polímeros recicláveis", "biocombustíveis", "fármacos quimioterápicos",
	"semicondutores de potência", "carros elétricos", "armazenamento de energia",
	"redes 5G", "edge computing", "internet das coisas", "smart grids",
	"reconhecimento de patentes via IA", "análise jurídica automatizada",
	"diagnóstico médico por imagem", "previsão climática",
	"agricultura de precisão", "irrigação inteligente",
	"detecção de fraudes financeiras", "sensores ambientais",
	"construção sustentável", "habitação social",
	"prótese ortopédica", "implante dentário", "stent cardíaco",
}

var patentApplicants = []string{
	"Universidade Federal de Ouro Preto",
	"Universidade Federal de Minas Gerais",
	"Universidade de São Paulo",
	"UNICAMP",
	"Universidade Federal do Rio de Janeiro",
	"Petrobras S.A.",
	"Embraer S.A.",
	"WEG Equipamentos Elétricos S.A.",
	"Vale S.A.",
	"Embrapa",
	"Fundação Oswaldo Cruz",
	"Argos Tech Ltda",
	"Innovabra Indústria Ltda",
	"NanoBR Pesquisa e Desenvolvimento Ltda",
	"BioCerrado S.A.",
}

func seedPatents(ctx context.Context, db *sql.DB, rng *rand.Rand) (int, error) {
	const q = `
		INSERT INTO patents (
			application_number, title, abstract, applicant, inventors,
			filing_date, publication_date, ipc_category, ipc_code, rpi_issue, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (application_number) DO NOTHING`

	count := 0
	for i := 0; i < 100; i++ {
		tpl := patentTemplates[rng.Intn(len(patentTemplates))]
		noun := patentNouns[rng.Intn(len(patentNouns))]
		applicant := patentApplicants[rng.Intn(len(patentApplicants))]

		// Filing date 2018-2025
		year := 2018 + rng.Intn(8)
		month := 1 + rng.Intn(12)
		day := 1 + rng.Intn(28)
		filing := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

		// 6 months later, publication
		pubDate := filing.AddDate(0, 6, 0)

		appNum := fmt.Sprintf("BR10%d%05d", year, 10000+i)
		title := fmt.Sprintf(tpl.titleFmt, noun)
		rpi := fmt.Sprintf("%d-%03d", year, 1+rng.Intn(52))

		// Most classified, a few pending/failed
		status := "classified"
		r := rng.Intn(100)
		if r < 5 {
			status = "failed"
		} else if r < 15 {
			status = "pending"
		}

		inventors := pickInventors(rng)

		_, err := db.ExecContext(ctx, q,
			appNum, title, tpl.abstract, applicant, pgArray(inventors),
			filing, pubDate, tpl.cat, tpl.ipcCode, rpi, status,
		)
		if err != nil {
			return count, fmt.Errorf("insert patent %s: %w", appNum, err)
		}
		count++
	}
	return count, nil
}

// ─── Trademarks ───────────────────────────────────────────────────────────────

var brandSeeds = []struct {
	name    string
	owner   string
	classes []int
}{
	{"ARGOS INTELLIGENCE", "Argos Tech Ltda", []int{9, 42}},
	{"VEGABRAS", "Bebidas Tropicais S.A.", []int{32, 33}},
	{"NANOFERRO", "NanoBR Pesquisa Ltda", []int{1, 6}},
	{"MINERAÇÃO VERDE", "Vale Sustentável S.A.", []int{6, 40}},
	{"BIOSENSOR PLUS", "BioCerrado S.A.", []int{9, 10}},
	{"AGROSMART", "AgroTech Brasil Ltda", []int{7, 9, 42}},
	{"PHARMABRAS", "PharmaBR Pesquisa", []int{5, 10}},
	{"SOLARDOMUS", "Energia Limpa S.A.", []int{9, 11, 42}},
	{"PRECISA SAÚDE", "MedCenter Brasil", []int{44, 5}},
	{"DENTECH", "Implantes Brasil Ltda", []int{10}},
	{"AUTOMOTIVA BR", "Embraer Components", []int{12}},
	{"DUROFLEX", "Polímeros do Brasil S.A.", []int{17}},
	{"ARGOLABS", "Argos Labs", []int{42, 35}},
	{"BIOENERGY", "Energia Renovável BR", []int{4, 40}},
	{"VITALBRAS", "Vital Indústria Farmacêutica", []int{5}},
	{"TECHCARE", "TechCare Saúde", []int{44, 42}},
	{"GREENWAY LOGÍSTICA", "Logística Sustentável S.A.", []int{39}},
	{"INOVAUFOP", "Universidade Federal de Ouro Preto", []int{41, 42}},
	{"MINASCHEM", "Química Minas Gerais Ltda", []int{1, 2}},
	{"AURUM EXTRACTOR", "Mineração Aurum S.A.", []int{1, 40}},
	{"NANOCOAT", "Revestimentos Avançados Ltda", []int{1, 2}},
	{"GERMINI", "GerminAg Ltda", []int{31}},
	{"PRATICUS", "Praticus Alimentos", []int{29, 30}},
	{"AQUATECH BRASIL", "Aquaculture Tech BR", []int{31, 42}},
	{"CIDADE INTELIGENTE", "SmartCity Solutions", []int{9, 42}},
	{"WEGSMART", "WEG Equipamentos Elétricos S.A.", []int{7, 9}},
	{"PETRÓLEO+", "Petrobras S.A.", []int{4, 1}},
	{"BANCO ARGOS", "Argos Holding Financeira", []int{36}},
	{"EDUTECH UFOP", "UFOP Extensão", []int{41}},
	{"BIOSOFT", "Biocomp Software", []int{9, 42}},
}

func seedTrademarks(ctx context.Context, db *sql.DB, rng *rand.Rand) (int, error) {
	const q = `
		INSERT INTO trademarks (
			process_number, name, normalized_name, kind, owner,
			nice_classes, status, filing_date, publication_date, granted_date
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (process_number) DO NOTHING`

	statuses := []string{"granted", "granted", "granted", "filed", "published", "denied"}
	kinds := []string{"nominative", "nominative", "mixed", "figurative"}
	count := 0

	for i, b := range brandSeeds {
		filing := time.Date(2019+rng.Intn(6), time.Month(1+rng.Intn(12)), 1+rng.Intn(28), 0, 0, 0, 0, time.UTC)
		pub := filing.AddDate(0, 3, 0)
		status := statuses[rng.Intn(len(statuses))]

		var granted sql.NullTime
		if status == "granted" {
			granted = sql.NullTime{Time: pub.AddDate(0, 8, 0), Valid: true}
		}

		processNum := fmt.Sprintf("9%08d", 10000000+i)
		normalized := strings.ToUpper(stripAccents(b.name))

		_, err := db.ExecContext(ctx, q,
			processNum, b.name, normalized, kinds[rng.Intn(len(kinds))], b.owner,
			pgIntArray(b.classes), status, filing, pub, granted,
		)
		if err != nil {
			return count, fmt.Errorf("insert trademark %s: %w", b.name, err)
		}
		count++
	}
	return count, nil
}

// ─── Disputes ─────────────────────────────────────────────────────────────────

var disputeSeeds = []struct {
	caseNum, title, summary, kind, status string
}{
	{
		"ARB-2025-001",
		"Conflito de marca: VEGABRAS vs. VEGA NATURAL",
		"Alegação de imitação fonética e visual em produtos da classe 32 (bebidas).",
		"trademark_infringement", "in_review",
	},
	{
		"ARB-2025-002",
		"Disputa de autoria: Processo de lixiviação para lítio",
		"Discussão sobre coautoria entre pesquisadores da UFOP e parceiro industrial.",
		"authorship", "awaiting_info",
	},
	{
		"ARB-2025-003",
		"Infração de patente: Sensor óptico de metais pesados",
		"Empresa concorrente alegadamente utilizando tecnologia patenteada sem licença.",
		"patent_infringement", "open",
	},
	{
		"ARB-2024-018",
		"Resolução de licenciamento: TT BIOSENSOR PLUS",
		"Negociação de royalties para extensão de contrato de transferência tecnológica.",
		"licensing", "resolved",
	},
	{
		"ARB-2024-022",
		"Marca AGROSMART: oposição de terceiro",
		"Oposição formal apresentada no INPI durante o período de publicação.",
		"trademark_infringement", "escalated",
	},
}

func seedDisputes(ctx context.Context, db *sql.DB, rng *rand.Rand) (int, error) {
	const q = `
		INSERT INTO disputes (case_number, title, summary, kind, status, opened_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (case_number) DO NOTHING`

	count := 0
	for _, d := range disputeSeeds {
		opened := time.Now().AddDate(0, -rng.Intn(6), -rng.Intn(28))

		_, err := db.ExecContext(ctx, q,
			d.caseNum, d.title, d.summary, d.kind, d.status, opened,
		)
		if err != nil {
			return count, fmt.Errorf("insert dispute %s: %w", d.caseNum, err)
		}
		count++
	}
	return count, nil
}

// ─── UFOP Opportunities ───────────────────────────────────────────────────────

var ufopSeeds = []struct {
	title, abstract, dept string
	ipcCat                int
	ipcSuggestion         string
	level                 string
	piScore               float64
	simPct                int
	source                string
}{
	{
		"Processo de biorremediação de solos contaminados com metais pesados via consórcio microbiano",
		"Método biológico para tratamento de solos com metais pesados oriundos de atividades de mineração, utilizando consórcio de microrganismos selecionados. Apresenta eficiência superior a 85% na remoção de chumbo e arsênio.",
		"Departamento de Química — UFOP",
		2, "C — Química e Metalurgia", "high", 8.7, 22, "oai",
	},
	{
		"Sistema de controle inteligente para distribuição de energia em microrredes rurais",
		"Sistema embarcado de controle adaptativo para otimização de fluxo de energia em microrredes isoladas. Algoritmo proprietário reduz perdas em até 30% e melhora resiliência a falhas.",
		"Departamento de Engenharia Elétrica — UFOP",
		7, "H — Eletricidade", "high", 7.9, 35, "oai",
	},
	{
		"Pesquisadores da UFOP desenvolvem método inovador de síntese de nanomateriais para aplicações biomédicas",
		"Equipe do ICEB desenvolveu técnica de síntese de nanopartículas de óxido de ferro com alto grau de pureza e biocompatibilidade, abrindo caminho para aplicações em diagnóstico por imagem e terapia direcionada.",
		"Instituto de Ciências Exatas e Biológicas",
		2, "C — Química e Metalurgia", "medium", 5.4, 45, "portal",
	},
	{
		"Biossensor eletroquímico para detecção rápida de patógenos em água potável",
		"Sensor de baixo custo baseado em grafeno funcionalizado que detecta E. coli e outros patógenos em menos de 5 minutos com sensibilidade ppb.",
		"Departamento de Engenharia Ambiental — UFOP",
		6, "G — Física / TI", "high", 8.2, 18, "oai",
	},
	{
		"Novo método de lixiviação em pilha para mineração de lítio em pegmatitos",
		"Desenvolvimento de método hidrometalúrgico inovador para extração de lítio de pegmatitos da região de Araçuaí com eficiência 40% superior aos métodos convencionais.",
		"Engenharia de Minas — UFOP",
		2, "C — Química e Metalurgia", "high", 9.1, 14, "oai",
	},
	{
		"Algoritmo de aprendizado federado para otimização de redes elétricas rurais",
		"Protocolo de aprendizado de máquina federado para otimização de microgrids em comunidades rurais sem exposição de dados locais.",
		"Engenharia Elétrica — UFOP",
		6, "G — Física / TI", "medium", 5.8, 41, "lens",
	},
	{
		"Composição farmacêutica derivada de plantas do cerrado para tratamento de inflamação",
		"Extrato padronizado de Lychnophora ericoides com atividade anti-inflamatória comprovada em modelos pré-clínicos.",
		"Farmácia — UFOP",
		0, "A — Necessidades Humanas", "medium", 5.1, 52, "oai",
	},
	{
		"Sistema construtivo modular de baixo custo para habitação social",
		"Painéis pré-fabricados de concreto leve que reduzem custo de obra em 35% e tempo de construção em 50%.",
		"Engenharia Civil — UFOP",
		4, "E — Construção Civil", "medium", 4.8, 48, "oai",
	},
	{
		"Aplicativo móvel para mapeamento colaborativo de patrimônio histórico",
		"Plataforma colaborativa com IA para identificação e catalogação de patrimônio em cidades históricas.",
		"Turismo — UFOP",
		6, "G — Física / TI", "low", 2.9, 67, "portal",
	},
	{
		"Estudo sobre micropoluentes em águas superficiais da bacia do Rio Doce",
		"Caracterização química de contaminantes emergentes em águas da bacia, sem proposta tecnológica de remediação.",
		"Engenharia Ambiental — UFOP",
		2, "C — Química e Metalurgia", "low", 2.5, 71, "oai",
	},
	{
		"Análise estatística do impacto socioeconômico de programas universitários",
		"Estudo observacional de longo prazo sem aplicação tecnológica direta.",
		"Economia — UFOP",
		6, "G — Física / TI", "low", 1.8, 78, "oai",
	},
	{
		"Membrana cerâmica para tratamento de efluentes industriais ricos em metais",
		"Membrana híbrida cerâmica/polimérica com seletividade elevada para Cr(VI) e Pb(II).",
		"Engenharia Metalúrgica — UFOP",
		1, "B — Operações e Transportes", "high", 7.6, 29, "oai",
	},
	{
		"Catalisador heterogêneo para produção de biodiesel a partir de óleos residuais",
		"Catalisador sólido reutilizável com conversão > 95% e tolerância a ácidos graxos livres.",
		"Química — UFOP",
		2, "C — Química e Metalurgia", "high", 8.0, 25, "oai",
	},
	{
		"Modelagem preditiva de falhas em equipamentos de mineração via deep learning",
		"Rede neural temporal alimentada por sensores IoT que prevê falhas com 18 horas de antecedência.",
		"Engenharia de Computação — UFOP",
		6, "G — Física / TI", "high", 7.4, 38, "lens",
	},
	{
		"Liga metálica leve para componentes aeroespaciais",
		"Liga de alumínio-lítio modificada com resistência específica 25% superior às convencionais.",
		"Engenharia Metalúrgica — UFOP",
		2, "C — Química e Metalurgia", "high", 8.5, 21, "oai",
	},
	{
		"Sistema de monitoramento de barragens via sensores fibra óptica",
		"Rede de sensores distribuídos para monitoramento contínuo de integridade estrutural de barragens.",
		"Engenharia de Minas — UFOP",
		6, "G — Física / TI", "high", 7.8, 31, "oai",
	},
	{
		"Vacina recombinante contra arbovírus emergentes",
		"Plataforma vacinal baseada em VLPs produzidas em sistema bacteriano de baixo custo.",
		"Biologia — UFOP",
		0, "A — Necessidades Humanas", "high", 8.3, 27, "oai",
	},
	{
		"Sensor sem fio para monitoramento de qualidade de cimento durante hidratação",
		"Sensor piezelétrico incorporado que reporta evolução do processo de cura em tempo real.",
		"Engenharia Civil — UFOP",
		4, "E — Construção Civil", "medium", 5.0, 55, "portal",
	},
	{
		"Análise comparativa de métodos pedagógicos em engenharia",
		"Estudo educacional sem aplicação tecnológica direta.",
		"Educação — UFOP",
		0, "A — Necessidades Humanas", "low", 1.4, 82, "oai",
	},
	{
		"Processo de impressão 3D de próteses customizadas com material biocompatível",
		"Método de fabricação aditiva de próteses ortopédicas com PEEK reforçado para uso clínico.",
		"Engenharia de Materiais — UFOP",
		0, "A — Necessidades Humanas", "medium", 6.0, 42, "oai",
	},
}

func seedUFOP(ctx context.Context, db *sql.DB, rng *rand.Rand) (int, error) {
	const q = `
		INSERT INTO ufop_opportunities (
			source, external_id, title, authors, department, abstract, url,
			published_at, ipc_suggestion, ipc_category, opportunity_level,
			similarity_pct, pi_score, ai_analysis, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (source, external_id) DO NOTHING`

	count := 0
	for i, u := range ufopSeeds {
		authors := pickInventors(rng)
		pubDate := time.Now().AddDate(0, -rng.Intn(8), -rng.Intn(28))
		externalID := fmt.Sprintf("oai:repositorio.ufop.br:1/demo-%d", 1000+i)
		analysis := fmt.Sprintf(
			"Publicação \"%s\" classificada como %s potencial de PI na categoria %s "+
				"(PI Score: %.1f/10). Similaridade com anterioridades: %d%%.",
			u.title, u.level, u.ipcSuggestion, u.piScore, u.simPct,
		)
		status := "new"
		if rng.Intn(10) < 3 {
			status = "reviewed"
		}

		_, err := db.ExecContext(ctx, q,
			u.source, externalID, u.title, pgArray(authors), u.dept, u.abstract,
			"https://repositorio.ufop.br/handle/1/"+fmt.Sprint(1000+i),
			pubDate, u.ipcSuggestion, u.ipcCat, u.level,
			u.simPct, u.piScore, analysis, status,
		)
		if err != nil {
			return count, fmt.Errorf("insert ufop %s: %w", externalID, err)
		}
		count++
	}
	return count, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func pickInventors(rng *rand.Rand) []string {
	candidates := []string{
		"Carlos Henrique Silva", "Ana Paula Costa", "Mariana Lima",
		"Roberto Rocha", "Fernanda Oliveira", "Paulo Souza",
		"Marco Antônio Ferreira", "Carla Mendes", "Pedro Almeida",
		"Juliana Martins", "Bruno Santos", "Patricia Gomes",
	}
	n := 1 + rng.Intn(3)
	picked := map[string]bool{}
	out := make([]string, 0, n)
	for len(out) < n {
		c := candidates[rng.Intn(len(candidates))]
		if !picked[c] {
			picked[c] = true
			out = append(out, c)
		}
	}
	return out
}

// pgArray formats a []string as a Postgres text array literal: {"a","b"}.
// Escapes embedded quotes and backslashes per pq spec.
func pgArray(items []string) any {
	if len(items) == 0 {
		return "{}"
	}
	parts := make([]string, len(items))
	for i, s := range items {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		parts[i] = `"` + s + `"`
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// pgIntArray formats a []int as {1,2,3}.
func pgIntArray(items []int) any {
	if len(items) == 0 {
		return "{}"
	}
	parts := make([]string, len(items))
	for i, v := range items {
		parts[i] = fmt.Sprint(v)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// stripAccents lowercases and removes diacritics for normalized_name.
func stripAccents(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "ê", "e", "è", "e", "ë", "e",
		"í", "i", "î", "i", "ï", "i",
		"ó", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"Á", "A", "Ã", "A", "Â", "A",
		"É", "E", "Ê", "E", "Í", "I",
		"Ó", "O", "Ô", "O", "Õ", "O",
		"Ú", "U", "Ç", "C",
	)
	return replacer.Replace(s)
}
