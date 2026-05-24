"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useWatchlists } from "@/lib/hooks";
import { api } from "@/lib/api";
import type { Watchlist, WatchType } from "@/lib/types";
import { SkeletonList } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useToast } from "@/components/ui/toast";
import {
  Bell, Plus, Eye, Building2, Tag, FileText, Search,
  Trash2, RefreshCw, Zap,
} from "lucide-react";

const typeIcon: Record<WatchType, React.ReactNode> = {
  term:    <Search    size={14} className="text-indigo-400" />,
  brand:   <Tag       size={14} className="text-orange-400" />,
  company: <Building2 size={14} className="text-blue-400" />,
  patent:  <FileText  size={14} className="text-purple-400" />,
};

const typeLabel: Record<WatchType, string> = {
  term:    "Termo de busca",
  brand:   "Marca monitorada",
  company: "Empresa",
  patent:  "Número de patente",
};

export default function AlertasPage() {
  const { data, error, mutate, isLoading } = useWatchlists();
  const toast = useToast();
  const [newLabel, setNewLabel] = useState("");
  const [newType,  setNewType]  = useState<WatchType>("term");
  const [newQuery, setNewQuery] = useState("");
  const [newAutoDispute, setNewAutoDispute] = useState(false);
  const [newThreshold, setNewThreshold]     = useState(70);
  const [busy, setBusy]         = useState<number | "create" | "all" | null>(null);

  const isLive   = !error && !!data;
  const items: Watchlist[] = data?.items ?? [];

  const totalNew   = items.reduce((s, w) => s + w.new_count, 0);
  const alertCount = items.filter(w => w.status === "alert").length;

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!newLabel.trim()) return;
    setBusy("create");
    try {
      await api.watchlists.create({
        label: newLabel.trim(),
        watch_type: newType,
        query: newQuery.trim() || newLabel.trim(),
        auto_dispute: newAutoDispute,
        similarity_threshold: newThreshold,
      });
      setNewLabel(""); setNewQuery(""); setNewType("term");
      setNewAutoDispute(false); setNewThreshold(70);
      mutate();
      toast.success(
        "Alerta criado",
        newAutoDispute
          ? `"${newLabel.trim()}" — auto-dispute ativo (threshold ${newThreshold}%)`
          : `Monitorando "${newLabel.trim()}".`,
      );
    } catch (err) {
      toast.error("Falha ao criar alerta", err instanceof Error ? err.message : "Erro");
    } finally { setBusy(null); }
  }

  async function handleDelete(id: number) {
    setBusy(id);
    try {
      await api.watchlists.delete(id);
      mutate();
      toast.info("Alerta removido");
    } catch { toast.error("Falha ao remover"); }
    finally { setBusy(null); }
  }

  async function handleCheck(id: number) {
    setBusy(id);
    try {
      const updated = await api.watchlists.check(id);
      mutate();
      toast.success(
        updated.new_count > 0 ? `${updated.new_count} novo(s) match(es)` : "Verificação concluída",
        `Watchlist "${updated.label}"`,
      );
    } catch { toast.error("Falha na verificação"); }
    finally { setBusy(null); }
  }

  async function handleCheckAll() {
    setBusy("all");
    try {
      const r = await api.watchlists.checkAll();
      mutate();
      toast.success("Verificação completa", `${r.checked} alertas escaneados.`);
    } catch { toast.error("Falha na verificação"); }
    finally { setBusy(null); }
  }

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Bell size={22} />
            Alertas &amp; Watchlist
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Monitore concorrentes, termos-chave, marcas e patentes em tempo real
          </p>
        </div>
        <div className="flex items-center gap-2">
          {isLive ? (
            <span className="text-xs text-emerald-400 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
              persistido no banco
            </span>
          ) : (
            <span className="text-xs text-amber-400">backend offline</span>
          )}
          <Button variant="secondary" size="sm"
            onClick={handleCheckAll}
            disabled={busy === "all" || items.length === 0}>
            {busy === "all"
              ? <><div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Verificando…</>
              : <><Zap size={12} /> Verificar todos</>}
          </Button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <Card>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Total monitorado</p>
          <p className="text-2xl font-bold text-white">{items.length}</p>
        </Card>
        <Card>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Com novidades</p>
          <p className="text-2xl font-bold" style={{ color: alertCount > 0 ? "#fbbf24" : "#fff" }}>
            {alertCount}
          </p>
        </Card>
        <Card>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Novos itens encontrados</p>
          <p className="text-2xl font-bold text-white">{totalNew}</p>
        </Card>
      </div>

      {/* Add new alert */}
      <Card>
        <CardHeader>
          <CardTitle>Adicionar novo monitoramento</CardTitle>
        </CardHeader>
        <form onSubmit={handleCreate} className="space-y-3">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>
                Rótulo (mostrado na lista)
              </label>
              <input
                value={newLabel}
                onChange={e => setNewLabel(e.target.value)}
                placeholder="ex: Petrobras, Sistema IA, BR1020230..."
                required
                className="w-full px-4 py-2.5 rounded-lg text-sm outline-none"
                style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
              />
            </div>
            <div>
              <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>Tipo</label>
              <select value={newType} onChange={e => setNewType(e.target.value as WatchType)}
                className="w-full px-4 py-2.5 rounded-lg text-sm outline-none"
                style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}>
                <option value="term">Termo de busca</option>
                <option value="brand">Marca</option>
                <option value="company">Empresa</option>
                <option value="patent">Número de patente</option>
              </select>
            </div>
          </div>
          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>
              Query (opcional — usa o rótulo se vazio)
            </label>
            <input
              value={newQuery}
              onChange={e => setNewQuery(e.target.value)}
              placeholder="palavra-chave para o ILIKE no banco"
              className="w-full px-4 py-2.5 rounded-lg text-sm outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
            />
          </div>

          {/* Auto-dispute mode */}
          <div className="rounded-lg p-3" style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
            <label className="flex items-start gap-2 cursor-pointer">
              <input type="checkbox" checked={newAutoDispute}
                onChange={e => setNewAutoDispute(e.target.checked)}
                className="mt-0.5" />
              <div>
                <p className="text-sm text-white font-medium flex items-center gap-1">
                  ⚡ Detector proativo de infração
                </p>
                <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
                  Cria automaticamente uma disputa em draft quando achar match com similaridade alta
                  (padrão indústria: CompuMark, Markify).
                </p>
              </div>
            </label>
            {newAutoDispute && (
              <div className="mt-2 ml-6 flex items-center gap-2">
                <label className="text-xs" style={{ color: "var(--text-muted)" }}>Threshold:</label>
                <input type="range" min="30" max="100" step="5"
                  value={newThreshold}
                  onChange={e => setNewThreshold(Number(e.target.value))}
                  className="flex-1" />
                <span className="text-xs font-mono text-amber-300 w-10 text-right">{newThreshold}%</span>
              </div>
            )}
          </div>

          <Button type="submit" size="sm" disabled={busy === "create" || !newLabel.trim()}>
            {busy === "create"
              ? <><div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Criando…</>
              : <><Plus size={14} /> Criar alerta</>}
          </Button>
        </form>
      </Card>

      {/* Alert list */}
      <div className="space-y-3">
        <h2 className="text-sm font-semibold text-white">Monitoramentos ativos</h2>

        {isLoading && items.length === 0 && <SkeletonList count={3} />}

        {!isLoading && items.length === 0 && (
          <Card>
            <EmptyState
              icon={Bell}
              title="Nenhum monitoramento ainda"
              description="Crie o primeiro alerta no formulário acima — Argos vai varrer patentes e marcas em busca de matches."
              size="sm"
            />
          </Card>
        )}

        {items.map(alert => (
          <Card key={alert.id}
            style={{ borderColor: alert.status === "alert" ? "#f59e0b30" : "var(--border)" }}>
            <div className="flex items-center gap-4">
              <div className="p-2 rounded-lg shrink-0" style={{ background: "var(--surface-2)" }}>
                {typeIcon[alert.watch_type]}
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-0.5 flex-wrap">
                  <Badge variant="muted">{typeLabel[alert.watch_type]}</Badge>
                  {alert.status === "alert" && (
                    <Badge variant="warning">🔔 {alert.new_count} novos</Badge>
                  )}
                  {alert.status === "ok" && (
                    <Badge variant="success">✓ Sem novidades</Badge>
                  )}
                  {alert.query && alert.query !== alert.label && (
                    <span className="font-mono text-xs text-indigo-400">"{alert.query}"</span>
                  )}
                  {alert.auto_dispute && (
                    <Badge variant="info">⚡ auto-dispute {alert.similarity_threshold}%</Badge>
                  )}
                </div>
                <p className="text-sm font-medium text-white">{alert.label}</p>
                <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
                  {alert.last_check
                    ? `Última verificação: ${new Date(alert.last_check).toLocaleString("pt-BR")}`
                    : "Nunca verificado"}
                </p>
              </div>
              <div className="flex gap-2 items-center">
                <Button variant="ghost" size="sm"
                  onClick={() => handleCheck(alert.id)}
                  disabled={busy === alert.id}>
                  {busy === alert.id
                    ? <div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    : <RefreshCw size={12} />}
                  Verificar
                </Button>
                {alert.status === "alert" && (
                  <Button size="sm">
                    <Eye size={12} />
                    Ver matches
                  </Button>
                )}
                <Button variant="ghost" size="sm"
                  onClick={() => handleDelete(alert.id)}
                  disabled={busy === alert.id}
                  style={{ color: "#f87171" }}>
                  <Trash2 size={12} />
                </Button>
              </div>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
