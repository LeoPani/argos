// EmptyState — used in tables, lists, panels when there's no data yet
// or when a filter narrowed everything out.

import { LucideIcon, Inbox } from "lucide-react";
import { Button } from "./button";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
    icon?: LucideIcon;
  };
  size?: "sm" | "md" | "lg";
}

export function EmptyState({
  icon: Icon = Inbox,
  title,
  description,
  action,
  size = "md",
}: EmptyStateProps) {
  const padding = size === "sm" ? "py-6"  : size === "lg" ? "py-16" : "py-10";
  const iconSize = size === "sm" ? 24    : size === "lg" ? 44     : 32;

  return (
    <div className={`text-center ${padding} space-y-3`}>
      <div className="inline-flex items-center justify-center rounded-full p-3"
        style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
        <Icon size={iconSize} className="text-slate-500" />
      </div>
      <div>
        <p className="text-sm font-medium text-white">{title}</p>
        {description && (
          <p className="text-xs mt-1 max-w-md mx-auto" style={{ color: "var(--text-muted)" }}>
            {description}
          </p>
        )}
      </div>
      {action && (
        <Button size="sm" onClick={action.onClick}>
          {action.icon && <action.icon size={12} />}
          {action.label}
        </Button>
      )}
    </div>
  );
}
