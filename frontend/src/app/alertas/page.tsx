"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { mockAlerts } from "@/lib/mock-data";
import { formatDate } from "@/lib/utils";
import type { Alert } from "@/lib/types";
import { Bell, Plus, Eye, Building2, Tag, FileText, Search } from "lucide-react";

const typeIcon: Record<Alert["type"], React.ReactNode> = {
  term: <Search size={14} className="text-indigo-400" />,
  brand: <Tag size={14} className="text-orange-400" />,
  company: <Building2 size={14} className="text-blue-400" />,
  patent: <FileText size={14} className="text-purple-400" />,
};

const typeLabel: Record<Alert["type"], string> = {
  term: "Termo de busca",
  brand: "Marca monitorada",
  company: "Empresa",
  patent: "Patente",
};

export default function AlertasPage() {
  const [alerts, setAlerts] = useState(mockAlerts);

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Bell size={22} />
            Alertas & Watchlist
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Monitore concorrentes, termos-chave, marcas e patentes em tempo real
          </p>
        </div>
        <Button size="sm">
          <Plus size={14} />
          Novo alerta
        </Button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <Card>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Total monitorado</p>
          <p className="text-2xl font-bold text-white">{alerts.length}</p>
        </Card>
        <Card>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Com novidades</p>
          <p className="text-2xl font-bold text-amber-400">{alerts.filter(a => a.status === "alert").length}</p>
        </Card>
        <Card>
          <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>Novos itens encontrados</p>
          <p className="text-2xl font-bold text-white">{alerts.reduce((s, a) => s + a.new_count, 0)}</p>
        </Card>
      </div>

      {/* Alert list */}
      <div className="space-y-3">
        {alerts.map(alert => (
          <Card key={alert.id}
            style={{ borderColor: alert.status === "alert" ? "#f59e0b30" : "var(--border)" }}>
            <div className="flex items-center gap-4">
              <div className="p-2 rounded-lg shrink-0" style={{ background: "var(--surface-2)" }}>
                {typeIcon[alert.type]}
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-0.5">
                  <Badge variant="muted">{typeLabel[alert.type]}</Badge>
                  {alert.status === "alert" && (
                    <Badge variant="warning">🔔 {alert.new_count} novos</Badge>
                  )}
                  {alert.status === "ok" && <Badge variant="success">✓ Sem novidades</Badge>}
                </div>
                <p className="text-sm font-medium text-white">{alert.label}</p>
                <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
                  Última verificação: {new Date(alert.last_check).toLocaleString("pt-BR")}
                </p>
              </div>
              <div className="flex gap-2">
                {alert.status === "alert" && (
                  <Button size="sm">
                    <Eye size={12} />
                    Ver novidades
                  </Button>
                )}
                <Button variant="ghost" size="sm"
                  onClick={() => setAlerts(prev => prev.filter(a => a.id !== alert.id))}>
                  Remover
                </Button>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Add new alert */}
      <Card>
        <CardHeader>
          <CardTitle>Adicionar novo monitoramento</CardTitle>
        </CardHeader>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>O que monitorar</label>
            <input
              placeholder="Nome da empresa, termo, marca..."
              className="w-full px-4 py-2.5 rounded-lg text-sm outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
            />
          </div>
          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>Tipo</label>
            <select className="w-full px-4 py-2.5 rounded-lg text-sm outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}>
              <option value="term">Termo de busca</option>
              <option value="brand">Marca</option>
              <option value="company">Empresa</option>
              <option value="patent">Número de patente</option>
            </select>
          </div>
        </div>
        <Button className="mt-4" size="sm">
          <Plus size={14} />
          Criar alerta
        </Button>
      </Card>
    </div>
  );
}
