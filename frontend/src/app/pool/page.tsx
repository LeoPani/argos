"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonKPI, SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useTTContracts, usePools, usePool, usePatents } from "@/lib/hooks";
import { api } from "@/lib/api";
import { formatBRL, formatDate } from "@/lib/utils";
import type {
  TTContract, LicenseKind, ContractStatus,
  PoolKind, Milestone,
} from "@/lib/types";
import {
  Briefcase, Plus, X, FileText, Coins, Layers,
  Building2, Calendar, Percent, RefreshCw, Trash2,
  CheckCircle2, ArrowRight,
} from "lucide-react";

type Tab = "contracts" | "pools";

// ─── labels ───────────────────────────────────────────────────────────────────

const licenseLabel: Record<LicenseKind, string> = {
  exclusive:     "Exclusiva",
  non_exclusive: "Não-exclusiva",
  sole:          "Única (sole)",
};

const contractStatusLabel: Record<ContractStatus, { label: string; variant: "info" | "warning" | "success" | "muted" | "danger" }> = {
  draft:       { label: "Rascunho",     variant: "muted"   },
  negotiating: { label: "Negociando",   variant: "warning" },
  active:      { label: "Ativo",        variant: "success" },
  expired:     { label: "Expirado",     variant: "muted"   },
  terminated:  { label: "Rescindido",   variant: "danger"  },
};

const poolKindLabel: Record<PoolKind, string> = {
  offensive:           "Ofensivo",
  defensive:           "Defensivo",
  standard_essential:  "SEP / FRAND",
};

// ─── main page ────────────────────────────────────────────────────────────────

export default function PoolPage() {
  const [tab, setTab] = useState<Tab>("contracts");

  return (
    <div className="p-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Briefcase size={22} />
          Pool de Patentes &amp; Contratos TT
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Gestão de contratos de transferência tecnológica e pools de patentes do NIT-UFOP
        </p>
      </div>

      <div className="flex gap-1 p-1 rounded-lg w-fit" style={{ background: "var(--surface)" }}>
        {(["contracts", "pools"] as Tab[]).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className="px-4 py-2 rounded-md text-sm transition-all"
            style={{
              background: tab === t ? "var(--accent)" : "transparent",
              color: tab === t ? "white" : "var(--text-muted)",
            }}>
            {t === "contracts"
              ? <><FileText size={13} className="inline mr-1.5" /> Contratos TT</>
              : <><Layers size={13} className="inline mr-1.5" /> Pools de Patentes</>}
          </button>
        ))}
      </div>

      {tab === "contracts" ? <ContractsTab /> : <PoolsTab />}
    </div>
  );
}

// ─── Contracts tab ────────────────────────────────────────────────────────────

function ContractsTab() {
  const { data, error, isLoading, mutate } = useTTContracts({ limit: "50" });
  const [showForm, setShowForm] = useState(false);

  const isLive   = !error && !!data;
  const loading  = isLoading && !data && !error;
  const items: TTContract[] = data?.items ?? [];

  const active = items.filter(c => c.status === "active").length;
  const totalRoyaltyFloor = items.reduce((s, c) => s + (c.status === "active" ? c.royalty_floor_annual : 0), 0);
  const totalUpfront = items.reduce((s, c) => s + c.upfront_fee, 0);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-white">Contratos TT</h2>
        <div className="flex gap-2 items-center">
          {isLive
            ? <span className="text-xs text-emerald-400">{data!.pagination.total} contratos</span>
            : <span className="text-xs text-amber-400">backend offline</span>}
          <Button variant="ghost" size="sm" onClick={() => mutate()}>
            <RefreshCw size={13} /> Atualizar
          </Button>
          <Button size="sm" onClick={() => setShowForm(s => !s)}>
            {showForm ? <X size={13} /> : <Plus size={13} />}
            {showForm ? "Cancelar" : "Novo contrato"}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {loading ? (
          <><SkeletonKPI /><SkeletonKPI /><SkeletonKPI /></>
        ) : (
          <>
            <KPI label="Contratos ativos"     value={active.toString()}              sub={`de ${items.length} totais`} color="#34d399" />
            <KPI label="Royalty floor anual"  value={formatBRL(totalRoyaltyFloor)}   sub="dos contratos ativos"        color="#6366f1" />
            <KPI label="Upfront acumulado"    value={formatBRL(totalUpfront)}        sub="signing fees"                color="#a855f7" />
          </>
        )}
      </div>

      {showForm && <NewTTContractForm onCreated={() => { setShowForm(false); mutate(); }} />}

      {loading && <SkeletonList count={3} />}
      {!loading && items.length === 0 && (
        <Card>
          <EmptyState
            icon={FileText}
            title="Nenhum contrato TT cadastrado"
            description="Crie o primeiro contrato vinculando uma patente do portfolio UFOP."
            size="sm"
          />
        </Card>
      )}

      <div className="space-y-3">
        {items.map(c => <ContractRow key={c.id} contract={c} onChanged={() => mutate()} />)}
      </div>
    </div>
  );
}

function ContractRow({ contract, onChanged }: { contract: TTContract; onChanged: () => void }) {
  const [expanded, setExpanded] = useState(false);
  const [busy, setBusy] = useState(false);
  const { label, variant } = contractStatusLabel[contract.status];

  async function transition(s: ContractStatus) {
    setBusy(true);
    try { await api.ttContracts.updateStatus(contract.id, s); onChanged(); }
    finally { setBusy(false); }
  }

  async function remove() {
    if (!confirm(`Remover contrato ${contract.contract_number}?`)) return;
    setBusy(true);
    try { await api.ttContracts.delete(contract.id); onChanged(); }
    finally { setBusy(false); }
  }

  const milestones: Milestone[] = Array.isArray(contract.milestones) ? contract.milestones : [];

  return (
    <Card style={{ borderColor: contract.status === "active" ? "#34d39930" : "var(--border)" }}>
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <p className="font-mono text-xs text-indigo-400">{contract.contract_number}</p>
            <Badge variant={variant}>{label}</Badge>
            <Badge variant="muted">{licenseLabel[contract.license_kind]}</Badge>
            {contract.sublicensable && <Badge variant="info">Sub-licenciável</Badge>}
            {contract.nit_approved && <Badge variant="success">✓ NIT</Badge>}
          </div>
          <p className="text-sm font-semibold text-white">
            <Building2 size={12} className="inline mr-1 text-slate-400" />
            {contract.licensor} → {contract.licensee}
          </p>
          {contract.field_of_use && (
            <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
              Campo: {contract.field_of_use}
            </p>
          )}
        </div>
      </div>

      <div className="grid grid-cols-4 gap-3 text-xs mb-3">
        <Metric icon={Percent}   label="Royalty"     value={`${contract.royalty_rate.toFixed(1)}%`}                  highlight />
        <Metric icon={Coins}     label="Floor anual" value={formatBRL(contract.royalty_floor_annual)} />
        <Metric icon={Coins}     label="Upfront"     value={formatBRL(contract.upfront_fee)} />
        <Metric icon={Building2} label="Inventores"  value={`${contract.inventor_share_pct}%`} />
      </div>

      <div className="flex items-center gap-3 text-xs mb-2" style={{ color: "var(--text-muted)" }}>
        <Calendar size={11} /> Território: <span className="text-white">{contract.territory}</span>
        {contract.signed_at  && <>· Assinado: <span className="text-white">{formatDate(contract.signed_at)}</span></>}
        {contract.expires_at && <>· Vence: <span className="text-white">{formatDate(contract.expires_at)}</span></>}
      </div>

      {expanded && (
        <div className="space-y-2 mb-3 pt-3" style={{ borderTop: "1px solid var(--border)" }}>
          {milestones.length > 0 && (
            <div>
              <p className="text-xs font-semibold text-white mb-1">Marcos</p>
              <div className="space-y-1">
                {milestones.map((m, i) => (
                  <div key={i} className="flex items-center justify-between text-xs p-2 rounded"
                    style={{ background: "var(--surface-2)" }}>
                    <span className="flex items-center gap-1.5">
                      {m.done
                        ? <CheckCircle2 size={11} className="text-emerald-400" />
                        : <div className="w-3 h-3 border rounded-full border-slate-600" />}
                      <span className="text-white">{m.label}</span>
                    </span>
                    <span style={{ color: "var(--text-muted)" }}>
                      {m.due_date && formatDate(m.due_date)}
                      {m.fee_brl ? ` · ${formatBRL(m.fee_brl)}` : ""}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
          {contract.notes && (
            <div>
              <p className="text-xs font-semibold text-white mb-1">Anotações</p>
              <p className="text-xs leading-relaxed" style={{ color: "var(--text-muted)" }}>{contract.notes}</p>
            </div>
          )}
        </div>
      )}

      <div className="flex gap-2 items-center flex-wrap">
        <Button variant="ghost" size="sm" onClick={() => setExpanded(e => !e)}>
          {expanded ? "Recolher" : "Detalhes"}
        </Button>
        {contract.status === "draft" && (
          <Button variant="secondary" size="sm" onClick={() => transition("negotiating")} disabled={busy}>
            <ArrowRight size={11} /> Iniciar negociação
          </Button>
        )}
        {contract.status === "negotiating" && (
          <Button variant="secondary" size="sm" onClick={() => transition("active")} disabled={busy}>
            <CheckCircle2 size={11} /> Ativar contrato
          </Button>
        )}
        {contract.status === "active" && (
          <Button variant="secondary" size="sm" onClick={() => transition("terminated")} disabled={busy}>
            Rescindir
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={remove} disabled={busy} style={{ color: "#f87171" }}>
          <Trash2 size={11} />
        </Button>
      </div>
    </Card>
  );
}

function NewTTContractForm({ onCreated }: { onCreated: () => void }) {
  const { data: patents } = usePatents({ limit: "100" });
  const [num, setNum]         = useState(`TT-UFOP-${new Date().getFullYear()}-${String(Math.floor(Math.random() * 900 + 100))}`);
  const [patentID, setPatentID] = useState<string>("");
  const [licensee, setLicensee] = useState("");
  const [cnpj, setCnpj]         = useState("");
  const [kind, setKind]         = useState<LicenseKind>("non_exclusive");
  const [territory, setTerritory] = useState("BR");
  const [field, setField]       = useState("");
  const [royalty, setRoyalty]   = useState("3.0");
  const [floor, setFloor]       = useState("0");
  const [upfront, setUpfront]   = useState("0");
  const [share, setShare]       = useState("33");
  const [notes, setNotes]       = useState("");
  const [busy, setBusy]         = useState(false);
  const [error, setError]       = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true); setError(null);
    try {
      await api.ttContracts.create({
        contract_number: num,
        patent_id: patentID ? Number(patentID) : undefined,
        licensee,
        licensee_cnpj: cnpj,
        license_kind: kind,
        territory,
        field_of_use: field,
        royalty_rate:        Number(royalty),
        royalty_floor_annual: Number(floor),
        upfront_fee:         Number(upfront),
        inventor_share_pct:  Number(share),
        notes,
      });
      onCreated();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Erro ao criar contrato");
    } finally { setBusy(false); }
  }

  return (
    <Card style={{ borderColor: "var(--accent)" }}>
      <CardHeader>
        <CardTitle>Novo contrato TT</CardTitle>
      </CardHeader>
      <form onSubmit={submit} className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <FieldL label="Número do contrato"><input value={num} onChange={e => setNum(e.target.value)} className="inp" required /></FieldL>
          <FieldL label="Patente (do portfolio UFOP)">
            <select value={patentID} onChange={e => setPatentID(e.target.value)} className="inp">
              <option value="">— escolher —</option>
              {patents?.items.map(p => (
                <option key={p.id} value={p.id}>
                  {p.application_number} · {p.title.substring(0, 50)}
                </option>
              ))}
            </select>
          </FieldL>
          <FieldL label="Licenciada *">
            <input value={licensee} onChange={e => setLicensee(e.target.value)} className="inp" required
              placeholder="ex: Innovabra Indústria Ltda" />
          </FieldL>
          <FieldL label="CNPJ da licenciada">
            <input value={cnpj} onChange={e => setCnpj(e.target.value)} className="inp"
              placeholder="12.345.678/0001-90" />
          </FieldL>
          <FieldL label="Tipo de licença">
            <select value={kind} onChange={e => setKind(e.target.value as LicenseKind)} className="inp">
              <option value="non_exclusive">Não-exclusiva</option>
              <option value="exclusive">Exclusiva</option>
              <option value="sole">Única (sole)</option>
            </select>
          </FieldL>
          <FieldL label="Território">
            <select value={territory} onChange={e => setTerritory(e.target.value)} className="inp">
              <option value="BR">Brasil</option>
              <option value="LATAM">América Latina</option>
              <option value="WORLD">Mundial</option>
            </select>
          </FieldL>
        </div>
        <FieldL label="Campo de uso (opcional, livre se vazio)">
          <input value={field} onChange={e => setField(e.target.value)} className="inp"
            placeholder="ex: Biotecnologia industrial" />
        </FieldL>
        <div className="grid grid-cols-4 gap-3">
          <FieldL label="Royalty (%)"><input type="number" step="0.1" min="0" value={royalty} onChange={e => setRoyalty(e.target.value)} className="inp" /></FieldL>
          <FieldL label="Floor anual (R$)"><input type="number" step="100" min="0" value={floor} onChange={e => setFloor(e.target.value)} className="inp" /></FieldL>
          <FieldL label="Upfront (R$)"><input type="number" step="100" min="0" value={upfront} onChange={e => setUpfront(e.target.value)} className="inp" /></FieldL>
          <FieldL label="Inventores (% — Lei 10.973)"><input type="number" min="0" max="50" value={share} onChange={e => setShare(e.target.value)} className="inp" /></FieldL>
        </div>
        <FieldL label="Anotações">
          <textarea value={notes} onChange={e => setNotes(e.target.value)} rows={2} className="inp resize-y"
            placeholder="ex: Royalty escalonado 3.5% nos 3 primeiros anos, 2.5% depois" />
        </FieldL>

        {error && <p className="text-xs" style={{ color: "#f87171" }}>{error}</p>}
        <Button type="submit" size="sm" disabled={busy || !licensee.trim()}>
          {busy ? "Criando…" : <><Plus size={13} /> Criar contrato</>}
        </Button>
      </form>

      <style jsx>{`
        .inp {
          width: 100%; padding: 0.5rem 0.75rem;
          background: var(--surface-2); border: 1px solid var(--border);
          border-radius: 0.5rem; color: white; font-size: 0.875rem; outline: none;
        }
        .inp:focus { border-color: var(--accent); }
      `}</style>
    </Card>
  );
}

// ─── Pools tab ────────────────────────────────────────────────────────────────

function PoolsTab() {
  const { data, mutate, isLoading } = usePools();
  const [showForm, setShowForm] = useState(false);
  const [selectedID, setSelectedID] = useState<number | null>(null);

  const items = data?.items ?? [];
  const loading = isLoading && !data;

  return (
    <div className="grid grid-cols-3 gap-6">
      <div className="col-span-1 space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-white">Pools ({items.length})</h2>
          <Button size="sm" onClick={() => setShowForm(s => !s)}>
            {showForm ? <X size={13} /> : <Plus size={13} />}
          </Button>
        </div>

        {showForm && (
          <NewPoolForm onCreated={(id) => { setShowForm(false); setSelectedID(id); mutate(); }} />
        )}

        {loading && <SkeletonList count={2} />}

        {!loading && items.length === 0 && !showForm && (
          <Card>
            <EmptyState
              icon={Layers}
              title="Nenhum pool criado"
              description="Pools agrupam patentes UFOP para licenciamento conjunto."
              size="sm"
            />
          </Card>
        )}

        {items.map(p => (
          <button key={p.id} onClick={() => setSelectedID(p.id)} className="w-full text-left">
            <Card style={{
              borderColor: selectedID === p.id ? "var(--accent)" : "var(--border)",
              cursor: "pointer", transition: "border-color 0.2s",
            }}>
              <div className="flex items-start justify-between gap-2 mb-1">
                <p className="text-sm font-semibold text-white">{p.name}</p>
                <Badge variant={p.status === "active" ? "success" : p.status === "forming" ? "warning" : "muted"}>
                  {p.status}
                </Badge>
              </div>
              <p className="text-xs mb-2" style={{ color: "var(--text-muted)" }}>
                {poolKindLabel[p.pool_kind]} · {p.royalty_rate}% · {p.duration_years}y
              </p>
              <span className="text-xs" style={{ color: "var(--text-muted)" }}>
                {p.members?.length ?? 0} patentes
              </span>
            </Card>
          </button>
        ))}
      </div>

      <div className="col-span-2">
        {selectedID
          ? <PoolDetail poolID={selectedID} onDeleted={() => { setSelectedID(null); mutate(); }} />
          : <div className="flex items-center justify-center h-64 rounded-xl"
              style={{ border: "1px dashed var(--border)" }}>
              <p className="text-sm" style={{ color: "var(--text-muted)" }}>
                Selecione um pool para gerenciar membros
              </p>
            </div>}
      </div>
    </div>
  );
}

function NewPoolForm({ onCreated }: { onCreated: (id: number) => void }) {
  const [name, setName]       = useState("");
  const [desc, setDesc]       = useState("");
  const [kind, setKind]       = useState<PoolKind>("offensive");
  const [royalty, setRoyalty] = useState("4.0");
  const [duration, setDuration] = useState("10");
  const [busy, setBusy]       = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const created = await api.pools.create({
        name, description: desc, pool_kind: kind,
        royalty_rate: Number(royalty),
        duration_years: Number(duration),
      });
      onCreated(created.id);
      setName(""); setDesc("");
    } finally { setBusy(false); }
  }

  return (
    <Card style={{ borderColor: "var(--accent)" }}>
      <form onSubmit={submit} className="space-y-2">
        <input value={name} onChange={e => setName(e.target.value)} required
          placeholder="Nome do pool"
          className="w-full px-3 py-2 rounded text-sm outline-none"
          style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }} />
        <textarea value={desc} onChange={e => setDesc(e.target.value)} rows={2}
          placeholder="Descrição (área tecnológica, racional)"
          className="w-full px-3 py-2 rounded text-sm outline-none resize-y"
          style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }} />
        <select value={kind} onChange={e => setKind(e.target.value as PoolKind)}
          className="w-full px-3 py-2 rounded text-sm outline-none"
          style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}>
          <option value="offensive">Ofensivo (licenciamento conjunto)</option>
          <option value="defensive">Defensivo (proteção mútua)</option>
          <option value="standard_essential">SEP / FRAND</option>
        </select>
        <div className="flex gap-2">
          <input type="number" step="0.1" min="0" value={royalty} onChange={e => setRoyalty(e.target.value)}
            placeholder="Royalty %"
            className="flex-1 px-3 py-2 rounded text-sm outline-none"
            style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }} />
          <input type="number" min="1" value={duration} onChange={e => setDuration(e.target.value)}
            placeholder="Anos"
            className="w-20 px-3 py-2 rounded text-sm outline-none"
            style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }} />
        </div>
        <Button type="submit" size="sm" disabled={busy || !name.trim()}>
          {busy ? "Criando…" : <><Plus size={11} /> Criar pool</>}
        </Button>
      </form>
    </Card>
  );
}

function PoolDetail({ poolID, onDeleted }: { poolID: number; onDeleted: () => void }) {
  const { data: pool, mutate } = usePool(poolID);
  const { data: patentsData }  = usePatents({ limit: "100" });
  const [patentID, setPatentID] = useState("");
  const [share, setShare]       = useState("20");
  const [busy, setBusy]         = useState(false);

  if (!pool) return <Card><p className="text-xs text-center py-4" style={{ color: "var(--text-muted)" }}>Carregando…</p></Card>;

  async function addMember(e: React.FormEvent) {
    e.preventDefault();
    if (!patentID) return;
    setBusy(true);
    try {
      await api.pools.addMember(poolID, { patent_id: Number(patentID), share_pct: Number(share) });
      mutate();
      setPatentID("");
    } finally { setBusy(false); }
  }

  async function removeMember(pid: number) {
    setBusy(true);
    try { await api.pools.removeMember(poolID, pid); mutate(); }
    finally { setBusy(false); }
  }

  async function deletePool() {
    if (!confirm(`Deletar pool "${pool!.name}"?`)) return;
    setBusy(true);
    try { await api.pools.delete(poolID); onDeleted(); }
    finally { setBusy(false); }
  }

  const members    = pool.members ?? [];
  const totalShare = members.reduce((s, m) => s + m.share_pct, 0);

  const inPool = new Set(members.map(m => m.patent_id));
  const availablePatents = patentsData?.items.filter(p => !inPool.has(p.id)) ?? [];

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>{pool.name}</CardTitle>
          <div className="flex gap-2">
            <Badge variant="muted">{poolKindLabel[pool.pool_kind]}</Badge>
            <Button variant="ghost" size="sm" onClick={deletePool} style={{ color: "#f87171" }}>
              <Trash2 size={11} />
            </Button>
          </div>
        </CardHeader>
        <p className="text-sm" style={{ color: "var(--text-muted)" }}>{pool.description}</p>

        <div className="grid grid-cols-4 gap-3 mt-3 text-xs">
          <Metric icon={Percent}   label="Royalty agregado" value={`${pool.royalty_rate}%`}       highlight />
          <Metric icon={Calendar}  label="Duração"          value={`${pool.duration_years} anos`} />
          <Metric icon={Building2} label="Admin"            value={pool.administrator}            />
          <Metric icon={Layers}    label="Status"           value={pool.status}                   />
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>
            Patentes membros ({members.length})
            <span className="ml-2 text-xs font-normal"
              style={{ color: totalShare === 100 ? "#34d399" : "var(--text-muted)" }}>
              · Share total: {totalShare.toFixed(1)}% / 100%
            </span>
          </CardTitle>
        </CardHeader>

        {members.length === 0 && (
          <p className="text-xs py-2" style={{ color: "var(--text-muted)" }}>
            Adicione patentes do portfolio para compor o pool.
          </p>
        )}

        <div className="space-y-2 mb-3">
          {members.map(m => (
            <div key={m.id} className="flex items-center justify-between p-2 rounded"
              style={{ background: "var(--surface-2)" }}>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-xs text-indigo-400">{m.patent_number}</span>
                  <span className="text-xs text-white truncate max-w-[260px]">{m.patent_title}</span>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-sm font-mono font-semibold text-amber-400">
                  {m.share_pct.toFixed(1)}%
                </span>
                <Button variant="ghost" size="sm"
                  onClick={() => removeMember(m.patent_id)} disabled={busy}
                  style={{ color: "#f87171" }}>
                  <X size={10} />
                </Button>
              </div>
            </div>
          ))}
        </div>

        <form onSubmit={addMember} className="flex gap-2 items-end pt-3"
          style={{ borderTop: "1px solid var(--border)" }}>
          <div className="flex-1">
            <label className="text-xs" style={{ color: "var(--text-muted)" }}>Adicionar patente</label>
            <select value={patentID} onChange={e => setPatentID(e.target.value)}
              className="w-full px-2 py-1.5 rounded text-xs outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}>
              <option value="">— escolher —</option>
              {availablePatents.map(p => (
                <option key={p.id} value={p.id}>
                  {p.application_number} · {p.title.substring(0, 50)}
                </option>
              ))}
            </select>
          </div>
          <div className="w-24">
            <label className="text-xs" style={{ color: "var(--text-muted)" }}>Share %</label>
            <input type="number" step="0.5" min="0" max="100"
              value={share} onChange={e => setShare(e.target.value)}
              className="w-full px-2 py-1.5 rounded text-xs outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }} />
          </div>
          <Button type="submit" size="sm" disabled={busy || !patentID}>
            <Plus size={11} /> Adicionar
          </Button>
        </form>
        {totalShare > 100 && (
          <p className="text-xs mt-2" style={{ color: "#f87171" }}>
            ⚠ Share total ultrapassa 100% — ajuste antes de ativar.
          </p>
        )}
      </Card>
    </div>
  );
}

// ─── small components ─────────────────────────────────────────────────────────

function KPI({ label, value, sub, color }: { label: string; value: string; sub?: string; color: string }) {
  return (
    <Card>
      <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
      <p className="text-2xl font-bold text-white">{value}</p>
      {sub && <p className="text-xs mt-1" style={{ color }}>{sub}</p>}
    </Card>
  );
}

function Metric({ icon: Icon, label, value, highlight }: {
  icon: typeof Coins; label: string; value: string; highlight?: boolean;
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon size={11} className={highlight ? "text-amber-400" : "text-slate-500"} />
      <div>
        <p className="text-xs" style={{ color: "var(--text-muted)" }}>{label}</p>
        <p className={`text-sm ${highlight ? "text-amber-300 font-semibold" : "text-white"}`}>{value}</p>
      </div>
    </div>
  );
}

function FieldL({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>{label}</label>
      {children}
    </div>
  );
}
