import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Activity, Eye, EyeOff, Globe2, Loader2, Radio, RefreshCw, Server } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ProxyRow } from "@/components/ProxyRow";
import { PROXY_GRID, fmtTime } from "@/lib/utils";
import { usePrivacy } from "@/lib/privacy";
import type { Snapshot } from "@/lib/types";

const empty: Snapshot = {
  subscription: { ok: false, hostCount: 0, fetchedAt: "", userAgent: "" },
  geoRunning: false,
  pingRunning: false,
  refreshingAll: false,
  proxies: [],
};

export default function App() {
  const [snap, setSnap] = useState<Snapshot>(empty);
  const [error, setError] = useState<string | null>(null);
  const [openId, setOpenId] = useState<string | null>(null);
  const [pendingAll, setPendingAll] = useState(false);
  const [pendingIds, setPendingIds] = useState<Set<string>>(new Set());
  const { redact, toggle: togglePrivacy } = usePrivacy();

  useEffect(() => {
    let alive = true;
    const tick = async () => {
      try {
        const res = await fetch("/api/status");
        if (!res.ok) throw new Error(`status ${res.status}`);
        const data = (await res.json()) as Snapshot;
        if (alive) {
          setSnap(data);
          setError(null);
          if (data.refreshingAll) setPendingAll(false);
          setPendingIds((prev) => {
            if (prev.size === 0) return prev;
            const next = new Set(prev);
            for (const id of prev) {
              const p = data.proxies.find((x) => x.id === id);
              if (!p || p.refreshing) next.delete(id);
            }
            return next.size === prev.size ? prev : next;
          });
        }
      } catch (e) {
        if (alive) setError(e instanceof Error ? e.message : "fetch failed");
      }
    };
    tick();
    const id = setInterval(tick, 4000);
    return () => {
      alive = false;
      clearInterval(id);
    };
  }, []);

  useEffect(() => {
    if (pendingIds.size === 0) return;
    const t = window.setTimeout(() => setPendingIds(new Set()), 15000);
    return () => window.clearTimeout(t);
  }, [pendingIds]);

  const upCount = useMemo(
    () => snap.proxies.filter((p) => p.ping?.up).length,
    [snap.proxies],
  );

  const refreshingAll = pendingAll || !!snap.refreshingAll;

  async function refreshAll() {
    setPendingAll(true);
    try {
      const res = await fetch("/api/refresh", { method: "POST" });
      if (!res.ok) throw new Error(`refresh ${res.status}`);
    } catch (e) {
      setPendingAll(false);
      setError(e instanceof Error ? e.message : "refresh failed");
    }
  }

  async function refreshOne(id: string) {
    setPendingIds((prev) => new Set(prev).add(id));
    try {
      const res = await fetch("/api/refresh/proxy", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      });
      if (!res.ok) throw new Error(`refresh ${res.status}`);
    } catch (e) {
      setPendingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      setError(e instanceof Error ? e.message : "refresh failed");
    }
  }

  return (
    <div className="mx-auto max-w-[1400px] px-4 py-8">
      <header className="mb-8 flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
        <div>
          <p className="font-mono text-[11px] uppercase tracking-[0.28em] text-primary">
            exit telemetry
          </p>
          <h1 className="mt-1 text-3xl font-semibold tracking-tight md:text-4xl">
            xray-geocheck
          </h1>
          <p className="mt-2 max-w-xl text-sm text-muted-foreground">
            Latency, geolocation consensus and reputation for every subscription
            egress IP. Ping is an HTTP GET through the local SOCKS5 fronted by
            xray-core.
          </p>
        </div>
        <div className="flex flex-wrap items-end gap-2">
          <Stat
            icon={<Server className="h-3.5 w-3.5" />}
            label="hosts"
            value={String(snap.subscription.hostCount || snap.proxies.length)}
          />
          <Stat
            icon={<Radio className="h-3.5 w-3.5" />}
            label="up"
            value={`${upCount}/${snap.proxies.length || 0}`}
          />
          <Stat
            icon={<Activity className="h-3.5 w-3.5" />}
            label="last ping"
            value={fmtTime(snap.lastPingAt)}
          />
          <Stat
            icon={<Globe2 className="h-3.5 w-3.5" />}
            label="last geo"
            value={fmtTime(snap.lastGeoAt)}
          />
          <Button
            type="button"
            variant="outline"
            onClick={togglePrivacy}
            title="Hide ASN, org, server name and identity. Exit IPs stay masked either way."
          >
            {redact ? <Eye className="mr-2 h-4 w-4" /> : <EyeOff className="mr-2 h-4 w-4" />}
            {redact ? "Show details" : "Hide details"}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => void refreshAll()}
            disabled={refreshingAll}
            title="Re-fetch subscription, then ping and geocheck every proxy"
          >
            {refreshingAll ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="mr-2 h-4 w-4" />
            )}
            Refresh all
          </Button>
        </div>
      </header>

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Badge
          className={
            snap.subscription.ok
              ? "border-primary/40 bg-primary/10 text-primary"
              : "border-destructive/40 bg-destructive/10 text-destructive"
          }
        >
          {snap.subscription.ok ? "subscription ok" : "subscription error"}
        </Badge>
        {snap.pingRunning && (
          <Badge className="border-border bg-secondary text-muted-foreground">ping sweep</Badge>
        )}
        {snap.geoRunning && (
          <Badge className="border-border bg-secondary text-muted-foreground">geo sweep</Badge>
        )}
        {refreshingAll && (
          <Badge className="border-primary/40 bg-primary/10 text-primary">refreshing all</Badge>
        )}
        {snap.subscription.userAgent && (
          <Badge className="border-border bg-transparent text-muted-foreground">
            ua {snap.subscription.userAgent}
          </Badge>
        )}
        {error && <span className="text-xs text-destructive">{error}</span>}
        {snap.subscription.error && (
          <span className="text-xs text-destructive">{snap.subscription.error}</span>
        )}
      </div>

      <div className="overflow-x-auto rounded-xl border border-border bg-card/80 backdrop-blur">
        <div className="min-w-[1150px]">
        <div className={`grid ${PROXY_GRID} gap-2 border-b border-border px-4 py-2 font-mono text-[10px] uppercase tracking-wider text-muted-foreground`}>
          <div>proxy</div>
          <div>ping</div>
          <div>exit ip</div>
          <div>identity</div>
          <div>cc</div>
          <div>risk</div>
          <div>type</div>
          <div />
        </div>
        {snap.proxies.length === 0 ? (
          <div className="px-4 py-16 text-center text-sm text-muted-foreground">
            Waiting for subscription hosts…
          </div>
        ) : (
          snap.proxies.map((p) => (
            <ProxyRow
              key={p.id}
              proxy={{
                ...p,
                refreshing: p.refreshing || pendingIds.has(p.id) || refreshingAll,
              }}
              open={openId === p.id}
              onToggle={() => setOpenId(openId === p.id ? null : p.id)}
              onRefresh={() => void refreshOne(p.id)}
            />
          ))
        )}
        </div>
      </div>
    </div>
  );
}

function Stat({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="min-w-[120px] rounded-lg border border-border bg-card/70 px-3 py-2">
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className="mt-1 font-mono text-sm text-foreground">{value}</div>
    </div>
  );
}
