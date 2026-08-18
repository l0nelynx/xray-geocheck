import { ChevronDown, Loader2, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Flag, FlagText } from "@/components/Flag";
import { GeoDetail } from "@/components/GeoDetail";
import { PROXY_GRID, fmtMs } from "@/lib/utils";
import { Sensitive } from "@/lib/privacy";
import type { ProxyStatus } from "@/lib/types";

export function ProxyRow({
  proxy,
  open,
  onToggle,
  onRefresh,
}: {
  proxy: ProxyStatus;
  open: boolean;
  onToggle: () => void;
  onRefresh: () => void;
}) {
  const rep = proxy.geo?.report?.reputation;
  const ident = proxy.geo?.report?.identity;
  const up = proxy.ping?.up;
  const cc = rep?.country_code || ident?.as_country;
  const identityLabel = ident ? ident.as_name || ident.org || "—" : proxy.geo?.error || "awaiting geo";
  return (
    <div className="border-b border-border last:border-0">
      <div
        role="button"
        tabIndex={0}
        onClick={onToggle}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onToggle();
          }
        }}
        className={`grid w-full ${PROXY_GRID} cursor-pointer items-center gap-2 px-4 py-3 text-left hover:bg-secondary/40`}
      >
        <div className="min-w-0">
          <div className="truncate font-medium">
            <FlagText text={proxy.remarks} className="max-w-full" />
          </div>
          <div className="truncate font-mono text-[11px] text-muted-foreground">
            {proxy.protocol} · <Sensitive>{proxy.address || proxy.socksAddr}</Sensitive>
          </div>
        </div>
        <div>
          {proxy.ping ? (
            <span className={up ? "font-mono text-primary" : "font-mono text-destructive"}>
              {up ? fmtMs(proxy.ping.rttMs) : "down"}
            </span>
          ) : (
            <span className="font-mono text-muted-foreground">…</span>
          )}
        </div>
        <div className="truncate font-mono text-xs">{ident?.ipv4 || "—"}</div>
        <div className="min-w-0 truncate text-xs text-muted-foreground">
          {ident ? <Sensitive>{ident.as_name || ident.org || "—"}</Sensitive> : identityLabel}
        </div>
        <div className="flex items-center gap-1.5 font-mono text-xs">
          {cc ? (
            <>
              <Flag code={cc} />
              <span>{cc}</span>
            </>
          ) : (
            "—"
          )}
        </div>
        <div>
          {rep?.risk != null ? (
            <RiskMeter risk={rep.risk} />
          ) : (
            <span className="font-mono text-muted-foreground">—</span>
          )}
        </div>
        <div className="flex items-center justify-between gap-1">
          {rep?.type ? (
            <Badge className="border-border bg-secondary/50 text-foreground">{rep.type}</Badge>
          ) : (
            <span className="font-mono text-muted-foreground">—</span>
          )}
          <ChevronDown className={`h-4 w-4 text-muted-foreground transition ${open ? "rotate-180" : ""}`} />
        </div>
        <button
          type="button"
          title="Refresh this proxy"
          disabled={proxy.refreshing}
          onClick={(e) => {
            e.stopPropagation();
            onRefresh();
          }}
          className="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground disabled:pointer-events-none disabled:opacity-50"
        >
          {proxy.refreshing ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <RefreshCw className="h-3.5 w-3.5" />
          )}
        </button>
      </div>
      {open && (
        <div className="border-t border-border bg-background/40 px-4 py-4">
          <GeoDetail proxy={proxy} />
        </div>
      )}
    </div>
  );
}

function RiskMeter({ risk }: { risk: number }) {
  const pct = Math.max(0, Math.min(100, risk));
  const color = pct >= 70 ? "bg-destructive" : pct >= 40 ? "bg-amber-400" : "bg-primary";
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-12 overflow-hidden rounded-full bg-muted">
        <div className={`h-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="font-mono text-xs">{pct}</span>
    </div>
  );
}
