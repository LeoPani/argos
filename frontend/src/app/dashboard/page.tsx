"use client";

import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { mockPatents, mockIpcDistribution, mockCostTimeline } from "@/lib/mock-data";
import { usePatents } from "@/lib/hooks";
import { formatDate, ipcLabel, IPC_COLORS } from "@/lib/utils";
import {
  BarChart, Bar, PieChart, Pie, Cell, AreaChart, Area,
  XAxis, YAxis, Tooltip, ResponsiveContainer,
} from "recharts";
import { FileText, Tag, AlertTriangle, Cpu } from "lucide-react";

const statusData = [
  { name: "Ativas", value: 72, color: "#34d399" },
  { name: "Oposição", value: 18, color: "#f59e0b" },
  { name: "Extintas", value: 10, color: "#64748b" },
];

export default function DashboardPage() {
  // Fetch real patents from the Go API; fall back to mock if offline
  const { data: patentsData } = usePatents({ limit: "10" });
  const patents = patentsData?.items ?? mockPatents;
  const totalPatents = patentsData?.pagination.total ?? 1204;

  const metrics = [
    { icon: FileText, label: "Patentes", value: totalPatents.toLocaleString("pt-BR"), delta: "↑ ao vivo", color: "#6366f1" },
    { icon: Tag, label: "Marcas", value: "893", delta: "↑ 5 hoje", color: "#8b5cf6" },
    { icon: AlertTriangle, label: "Conflitos", value: "12", delta: "↓ 2 hoje", color: "#f59e0b" },
    { icon: Cpu, label: "Classificadas por IA", value: patents.length > 0 ? Math.round(patents.filter(p => p.status === "classified").length / patents.length * 100) + "%" : "97%", delta: "BERT ativo", color: "#34d399" },
  ];

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">BI & Analytics</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Visão geral do sistema — dados INPI + classificação IA
        </p>
      </div>

      {/* Métricas */}
      <div className="grid grid-cols-4 gap-4">
        {metrics.map(({ icon: Icon, label, value, delta, color }: { icon: React.ElementType; label: string; value: string; delta: string; color: string }) => (
          <Card key={label}>
            <div className="flex items-start justify-between">
              <div>
                <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
                <p className="text-2xl font-bold text-white">{value}</p>
                <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{delta}</p>
              </div>
              <div className="p-2 rounded-lg" style={{ background: color + "20" }}>
                <Icon size={18} style={{ color }} />
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Gráficos linha 1 */}
      <div className="grid grid-cols-3 gap-4">
        {/* IPC Distribution */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>Patentes por categoria IPC</CardTitle>
          </CardHeader>
          <div className="space-y-2">
            {mockIpcDistribution.map(({ name, value, cat }, i) => (
              <div key={cat}>
                <div className="flex justify-between text-xs mb-1">
                  <span style={{ color: "var(--text-muted)" }}>{name}</span>
                  <span className="text-white font-medium">{value}%</span>
                </div>
                <div className="h-1.5 rounded-full" style={{ background: "var(--border)" }}>
                  <div
                    className="h-full rounded-full"
                    style={{ width: `${value}%`, background: IPC_COLORS[i] }}
                  />
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Status das Marcas */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>Marcas por status</CardTitle>
          </CardHeader>
          <div className="flex items-center gap-4">
            <ResponsiveContainer width={120} height={120}>
              <PieChart>
                <Pie data={statusData} cx="50%" cy="50%" innerRadius={35} outerRadius={55} dataKey="value" strokeWidth={0}>
                  {statusData.map((d, i) => <Cell key={i} fill={d.color} />)}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
            <div className="space-y-2">
              {statusData.map(d => (
                <div key={d.name} className="flex items-center gap-2 text-xs">
                  <div className="w-2 h-2 rounded-full" style={{ background: d.color }} />
                  <span style={{ color: "var(--text-muted)" }}>{d.name}</span>
                  <span className="text-white font-medium ml-auto">{d.value}%</span>
                </div>
              ))}
            </div>
          </div>
        </Card>

        {/* Custo anual projetado */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>Custo anual projetado</CardTitle>
          </CardHeader>
          <ResponsiveContainer width="100%" height={130}>
            <AreaChart data={mockCostTimeline}>
              <defs>
                <linearGradient id="costGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#6366f1" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="year" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} tickFormatter={v => `R$${(v / 1000).toFixed(0)}k`} />
              <Tooltip formatter={(v) => [`R$ ${Number(v).toLocaleString("pt-BR")}`, "Custo"]} />
              <Area type="monotone" dataKey="value" stroke="#6366f1" fill="url(#costGrad)" strokeWidth={2} />
            </AreaChart>
          </ResponsiveContainer>
        </Card>
      </div>

      {/* Últimas ingestões */}
      <Card>
        <CardHeader>
          <CardTitle>Últimas ingestões</CardTitle>
          <span className="text-xs" style={{ color: "var(--text-muted)" }}>
            {patentsData ? "✓ dados ao vivo da API Go" : "modo offline — dados de exemplo"}
          </span>
        </CardHeader>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: "1px solid var(--border)" }}>
                {["Nº Pedido", "Título", "Titular", "Categoria IPC", "Status", "Criado em"].map(h => (
                  <th key={h} className="text-left pb-2 pr-4 text-xs font-medium" style={{ color: "var(--text-muted)" }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {patents.map(p => (
                <tr key={p.id} style={{ borderBottom: "1px solid var(--border)" }}
                  className="hover:bg-white/5 transition-colors">
                  <td className="py-3 pr-4 font-mono text-xs text-indigo-400">{p.application_number}</td>
                  <td className="py-3 pr-4 text-white max-w-[200px] truncate">{p.title}</td>
                  <td className="py-3 pr-4" style={{ color: "var(--text-muted)" }}>{p.applicant}</td>
                  <td className="py-3 pr-4">
                    <Badge variant="default">{ipcLabel(p.ipc_category)}</Badge>
                  </td>
                  <td className="py-3 pr-4">
                    <Badge variant={p.status === "classified" ? "success" : p.status === "failed" ? "danger" : "warning"}>
                      {p.status}
                    </Badge>
                  </td>
                  <td className="py-3 text-xs" style={{ color: "var(--text-muted)" }}>
                    {formatDate(p.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}
