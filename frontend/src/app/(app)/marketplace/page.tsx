"use client";

// /marketplace — vitrine pública de patentes UFOP disponíveis para licenciamento.
// Inspirado em AUTM University Marketplace + EPO/Yet2.com tech offers.
// Pode ser apresentada a empresas SEM autenticação.

import { useState } from "react";
import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useMarketplace } from "@/lib/hooks";
import type { MarketplaceListing } from "@/lib/types";
import {
  Briefcase, Search, Filter, Mail, ExternalLink, User,
  Calendar, Award, Lock, Layers,
} from "lucide-react";

const ipcSections = [
  { letter: "A", name: "Necessidades Humanas" },
  { letter: "B", name: "Operações / Transportes" },
  { letter: "C", name: "Química / Metalurgia" },
  { letter: "D", name: "Têxteis / Papel" },
  { letter: "E", name: "Construção Civil" },
  { letter: "F", name: "Engenharia Mecânica" },
  { letter: "G", name: "Física / TI" },
  { letter: "H", name: "Eletricidade" },
];

const ipcColors: Record<string, string> = {
  A: "#ef4444", B: "#f59e0b", C: "#34d399", D: "#06b6d4",
  E: "#3b82f6", F: "#8b5cf6", G: "#a855f7", H: "#ec4899",
};

export default function MarketplacePage() {
  const [ipc, setIpc]       = useState<string>("");
  const [q, setQ]           = useState<string>("");
  const [pending, setPending] = useState<string>("");
  const { data, isLoading } = useMarketplace({ ipc, q, limit: 50 });

  function applySearch(e: React.FormEvent) {
    e.preventDefault();
    setQ(pending);
  }

  const items = data?.items ?? [];

  return (
    <div className="min-h-screen" style={{ background: "var(--bg)" }}>
      {/* Public header — no sidebar */}
      <header className="px-8 py-6"
        style={{ background: "linear-gradient(135deg, #6366f1 0%, #a855f7 100%)" }}>
        <div className="max-w-6xl mx-auto">
          <div className="flex items-center gap-3 mb-2">
            <Briefcase size={24} className="text-white" />
            <h1 className="text-2xl font-bold text-white">UFOP TT Marketplace</h1>
            <Badge variant="muted">Público</Badge>
          </div>
          <p className="text-sm text-indigo-100">
            Tecnologias disponíveis para licenciamento da Universidade Federal de Ouro Preto
          </p>
          <p className="text-xs text-indigo-200 mt-1 opacity-90">
            Catálogo parcial — ingestão Google Patents pausada por rate-limit
            (10 de 261 patentes UFOP carregadas). Lens.org desbloqueia o restante.
          </p>
        </div>
      </header>

      <div className="max-w-6xl mx-auto p-8 space-y-6">
        {/* Search & filter */}
        <Card>
          <form onSubmit={applySearch} className="flex gap-3 items-end">
            <div className="flex-1">
              <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>Buscar tecnologia</label>
              <div className="relative">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                <input
                  value={pending}
                  onChange={e => setPending(e.target.value)}
                  placeholder="ex: lítio, biossensor, energia solar…"
                  className="w-full pl-10 pr-4 py-2.5 rounded-lg text-sm outline-none"
                  style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
                />
              </div>
            </div>
            <Button type="submit" size="sm">Buscar</Button>
          </form>

          <div className="flex gap-2 mt-3 flex-wrap items-center">
            <Filter size={11} className="text-slate-500" />
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>Filtrar por área IPC:</span>
            <button onClick={() => setIpc("")}
              className="px-2.5 py-1 rounded-full text-xs transition-colors"
              style={{
                background: ipc === "" ? "var(--accent)" : "var(--surface-2)",
                color: ipc === "" ? "white" : "var(--text-muted)",
                border: "1px solid var(--border)",
              }}>
              Todas
            </button>
            {ipcSections.map(s => {
              const count = data?.by_ipc_category[s.letter] ?? 0;
              const active = ipc === s.letter;
              return (
                <button key={s.letter} onClick={() => setIpc(s.letter)}
                  className="px-2.5 py-1 rounded-full text-xs transition-colors"
                  style={{
                    background: active ? ipcColors[s.letter] : "var(--surface-2)",
                    color: active ? "white" : count > 0 ? "white" : "var(--text-muted)",
                    border: `1px solid ${active ? ipcColors[s.letter] : "var(--border)"}`,
                  }}
                  title={s.name}>
                  {s.letter}
                  {count > 0 && <span className="ml-1 opacity-70">({count})</span>}
                </button>
              );
            })}
          </div>
        </Card>

        {/* Stats banner */}
        {data && (
          <div className="flex items-center justify-between text-sm">
            <span style={{ color: "var(--text-muted)" }}>
              <span className="text-white font-semibold">{data.count}</span> tecnologias disponíveis
              {q && <span> · filtrado por &quot;{q}&quot;</span>}
              {ipc && <span> · seção IPC {ipc}</span>}
            </span>
            <a href="mailto:nit@ufop.edu.br" className="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1">
              <Mail size={11} /> NIT-UFOP — nit@ufop.edu.br
            </a>
          </div>
        )}

        {/* Listings */}
        {isLoading && <SkeletonList count={4} />}

        {!isLoading && items.length === 0 && (
          <Card>
            <EmptyState
              icon={Briefcase}
              title="Nenhuma tecnologia encontrada"
              description={q || ipc ? "Tente outro filtro ou termo." : "Verifique se as patentes UFOP estão cadastradas."}
            />
          </Card>
        )}

        <div className="grid grid-cols-1 gap-3">
          {items.map(it => <ListingCard key={it.patent_id} item={it} />)}
        </div>

        {/* Footer */}
        <footer className="text-center pt-6 mt-12 text-xs"
          style={{ borderTop: "1px solid var(--border)", color: "var(--text-muted)" }}>
          Powered by <span className="text-indigo-400">Argos IP Intelligence</span> ·
          Núcleo de Inovação Tecnológica — UFOP ·
          Lei n. 10.973/2004 (Marco Legal C&amp;T)
        </footer>
      </div>
    </div>
  );
}

function ListingCard({ item }: { item: MarketplaceListing }) {
  const color = ipcColors[item.ipc_letter] ?? "#6366f1";
  const available = item.non_exclusive_slots_available > 0;

  return (
    <Card style={{ borderColor: available ? color + "30" : "var(--border)" }}>
      <div className="flex items-start gap-4">
        <div className="p-3 rounded-xl shrink-0"
          style={{ background: color + "20", border: `1px solid ${color}40` }}>
          <span className="text-lg font-bold" style={{ color }}>{item.ipc_letter}</span>
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
            <p className="font-mono text-xs text-indigo-400">{item.application_number}</p>
            <Badge variant="muted">{item.ipc_name}</Badge>
            {available ? (
              <Badge variant="success">
                {item.non_exclusive_slots_available} vagas disponíveis
              </Badge>
            ) : (
              <Badge variant="muted">
                <Lock size={9} /> Exclusiva ativa
              </Badge>
            )}
            {item.existing_licensees > 0 && (
              <Badge variant="info">
                {item.existing_licensees} licenciado(s)
              </Badge>
            )}
          </div>

          <h3 className="text-base font-semibold text-white leading-snug">{item.title}</h3>

          <p className="text-sm mt-1 leading-relaxed line-clamp-2"
            style={{ color: "var(--text-muted)" }}>
            {item.abstract}
          </p>

          <div className="flex items-center gap-4 text-xs mt-3" style={{ color: "var(--text-muted)" }}>
            {item.filing_year > 0 && (
              <span className="flex items-center gap-1">
                <Calendar size={11} /> {item.filing_year}
              </span>
            )}
            {item.inventors.length > 0 && (
              <span className="flex items-center gap-1">
                <User size={11} /> {item.inventors[0]}
                {item.inventors.length > 1 && ` +${item.inventors.length - 1}`}
              </span>
            )}
            <span className="flex items-center gap-1">
              <Award size={11} className="text-emerald-400" /> {item.status}
            </span>
            <span className="ml-auto text-xs">
              <span style={{ color: "var(--text-muted)" }}>Sugestão:</span>{" "}
              <span className="text-amber-300 font-medium">{item.suggested_license_kind}</span>
            </span>
          </div>

          <div className="flex gap-2 mt-3">
            <Link href={`/patents/${item.patent_id}`}>
              <Button variant="ghost" size="sm">
                <Layers size={11} /> Ver detalhes técnicos
              </Button>
            </Link>
            {available && (
              <a href={`mailto:nit@ufop.edu.br?subject=Interesse em licenciamento — ${item.application_number}`}>
                <Button size="sm">
                  <Mail size={11} /> Manifestar interesse
                </Button>
              </a>
            )}
          </div>
        </div>
      </div>
    </Card>
  );
}
