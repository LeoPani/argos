"use client";

import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Settings, Database, Cpu, Link2, Bell } from "lucide-react";

export default function ConfigPage() {
  return (
    <div className="p-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Settings size={22} />
          Configurações
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Conexões, integrações e preferências do sistema
        </p>
      </div>

      {/* Connections */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database size={15} />
            Conexões
          </CardTitle>
        </CardHeader>
        <div className="space-y-3">
          {[
            { label: "API Go (Backend)", url: "http://localhost:8080", status: "ok" },
            { label: "BERT Classifier (FastAPI)", url: "http://localhost:8000", status: "ok" },
            { label: "PostgreSQL", url: "localhost:5432 · argos", status: "ok" },
            { label: "Polygon (Blockchain)", url: "polygon-mainnet", status: "pending" },
            { label: "Lens.org API", url: "api.lens.org", status: "pending" },
            { label: "INPI BDRPI", url: "buscaweb.inpi.gov.br", status: "pending" },
          ].map(conn => (
            <div key={conn.label} className="flex items-center justify-between p-3 rounded-lg"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
              <div>
                <p className="text-sm font-medium text-white">{conn.label}</p>
                <p className="text-xs font-mono" style={{ color: "var(--text-muted)" }}>{conn.url}</p>
              </div>
              <Badge variant={conn.status === "ok" ? "success" : "muted"}>
                {conn.status === "ok" ? "✓ Conectado" : "Não configurado"}
              </Badge>
            </div>
          ))}
        </div>
      </Card>

      {/* AI Settings */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Cpu size={15} />
            Configurações de IA
          </CardTitle>
        </CardHeader>
        <div className="space-y-3">
          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>Modelo BERT (classificação)</label>
            <div className="px-4 py-2.5 rounded-lg text-sm"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
              BERTimbau fine-tuned · 8 categorias IPC · argos_model/
            </div>
          </div>
          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>LLM (geração de insights)</label>
            <div className="flex gap-2">
              <input placeholder="Anthropic API Key..." className="flex-1 px-4 py-2.5 rounded-lg text-sm outline-none"
                style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
                type="password" />
              <Button variant="secondary" size="sm">Salvar</Button>
            </div>
          </div>
        </div>
      </Card>

      {/* Notifications */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bell size={15} />
            Notificações
          </CardTitle>
        </CardHeader>
        <div className="space-y-3">
          {[
            { label: "Alertas de prazo (30 dias antes)", enabled: true },
            { label: "Novas anterioridades em watchlist", enabled: true },
            { label: "Novas oportunidades UFOP detectadas", enabled: true },
            { label: "Relatório semanal de portfolio", enabled: false },
          ].map(n => (
            <div key={n.label} className="flex items-center justify-between">
              <span className="text-sm" style={{ color: "var(--text-muted)" }}>{n.label}</span>
              <div className={`w-9 h-5 rounded-full cursor-pointer transition-colors ${n.enabled ? "bg-indigo-500" : ""}`}
                style={{ background: n.enabled ? "var(--accent)" : "var(--border)" }}>
                <div className={`w-4 h-4 bg-white rounded-full mt-0.5 transition-transform ${n.enabled ? "translate-x-4" : "translate-x-0.5"}`} />
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* Version */}
      <div className="text-center text-xs" style={{ color: "var(--text-muted)" }}>
        Argos IP Intelligence · v0.1.0-alpha · Phase 1 ✅ Phase D (Frontend) ✅
      </div>
    </div>
  );
}
