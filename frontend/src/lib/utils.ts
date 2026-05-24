import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatBRL(value: number): string {
  return new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
  }).format(value);
}

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Intl.DateTimeFormat("pt-BR").format(new Date(iso));
}

export function daysUntil(iso: string | null | undefined): number | null {
  if (!iso) return null;
  const diff = new Date(iso).getTime() - Date.now();
  return Math.ceil(diff / (1000 * 60 * 60 * 24));
}

export function riskColor(score: number): string {
  if (score <= 3) return "text-emerald-400";
  if (score <= 6) return "text-amber-400";
  if (score <= 8) return "text-orange-400";
  return "text-red-400";
}

export function riskBg(score: number): string {
  if (score <= 3) return "bg-emerald-400";
  if (score <= 6) return "bg-amber-400";
  if (score <= 8) return "bg-orange-400";
  return "bg-red-500";
}

export function riskLabel(score: number): string {
  if (score <= 3) return "Baixo";
  if (score <= 6) return "Médio";
  if (score <= 8) return "Alto";
  return "Muito Alto";
}

export function ipcLabel(cat: number | null): string {
  const labels: Record<number, string> = {
    0: "Necessidades Humanas",
    1: "Operações / Transportes",
    2: "Química / Metalurgia",
    3: "Têxteis / Papel",
    4: "Construção Civil",
    5: "Engenharia Mecânica",
    6: "Física",
    7: "Eletricidade",
  };
  return cat !== null ? labels[cat] ?? `Categoria ${cat}` : "—";
}

export const IPC_COLORS = [
  "#6366f1", "#8b5cf6", "#a78bfa", "#60a5fa",
  "#34d399", "#fbbf24", "#f87171", "#fb923c",
];
