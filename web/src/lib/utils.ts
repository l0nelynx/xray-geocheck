import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export const PROXY_GRID =
  "grid-cols-[minmax(240px,1.6fr)_100px_150px_minmax(220px,1.2fr)_80px_100px_110px_44px]";

export function fmtMs(ms?: number | null) {
  if (ms == null || Number.isNaN(ms)) return "—";
  if (ms < 10) return `${ms.toFixed(1)} ms`;
  return `${Math.round(ms)} ms`;
}

export function fmtTime(iso?: string | null) {
  if (!iso) return "never";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "never";
  return d.toLocaleString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    month: "short",
    day: "2-digit",
  });
}

export type StatusTone = "ok" | "warn" | "err" | "neutral";

export function statusTone(value?: string | null): StatusTone {
  const v = (value || "").trim().toLowerCase();
  if (!v) return "neutral";
  if (["available", "ok", "clean", "reachable", "up"].includes(v)) return "ok";
  if (["blocked", "captive", "altered", "captive_portal", "degraded"].includes(v)) return "warn";
  if (["error", "unreachable", "fail", "failed", "timeout", "down"].includes(v)) return "err";
  return "neutral";
}

export function statusToneClass(tone: StatusTone) {
  switch (tone) {
    case "ok":
      return "border-primary/40 bg-primary/10 text-primary";
    case "warn":
      return "border-amber-400/40 bg-amber-400/10 text-amber-300";
    case "err":
      return "border-destructive/40 bg-destructive/10 text-destructive";
    default:
      return "border-border bg-secondary text-muted-foreground";
  }
}
