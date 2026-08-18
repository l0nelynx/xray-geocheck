import { type ReactNode } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { CountryLabel, Flag } from "@/components/Flag";
import { fmtMs, statusTone, statusToneClass } from "@/lib/utils";
import { usePrivacy } from "@/lib/privacy";
import type { GeoEndpoint, GeoValue, ProxyStatus } from "@/lib/types";

export function GeoDetail({ proxy }: { proxy: ProxyStatus }) {
  const { redact } = usePrivacy();
  const geo = proxy.geo;
  const report = geo?.report;

  if (!geo) {
    return <p className="text-sm text-muted-foreground">No geocheck result yet. The first sweep runs after startup.</p>;
  }
  if (!geo.ok || !report) {
    return (
      <p className="text-sm text-destructive">
        geocheck failed{geo.error ? `: ${geo.error}` : ""}
      </p>
    );
  }

  const ident = report.identity;
  const rep = report.reputation;

  return (
    <Tabs defaultValue="identity">
      <TabsList>
        <TabsTrigger value="identity">Identity</TabsTrigger>
        <TabsTrigger value="reputation">Reputation</TabsTrigger>
        <TabsTrigger value="consensus">Consensus</TabsTrigger>
        <TabsTrigger value="geo">Geo</TabsTrigger>
        <TabsTrigger value="portal">Connectivity checks</TabsTrigger>
        <TabsTrigger value="stash">Access</TabsTrigger>
        <TabsTrigger value="ai">AI endpoints</TabsTrigger>
        <TabsTrigger value="path">Path</TabsTrigger>
      </TabsList>

      <TabsContent value="identity">
        <dl className="grid grid-cols-2 gap-3 text-sm md:grid-cols-4">
          <KV k="IPv4" v={ident?.ipv4} mono sensitive />
          <KV k="IPv6" v={ident?.ipv6} mono sensitive />
          <KV k="ASN" v={ident?.asn != null ? `AS${ident.asn}` : undefined} mono sensitive />
          <KV k="AS name" v={ident?.as_name} sensitive />
          <KV k="Org" v={ident?.org} sensitive />
          <KV k="AS country" v={ident?.as_country ? <CountryLabel code={ident.as_country} /> : undefined} sensitive />
          <KV k="Resolver" v={report.transport?.resolver} sensitive />
          <KV k="Duration" v={report.duration_ms != null ? fmtMs(report.duration_ms) : undefined} />
        </dl>
        {!!report.findings?.length && !redact && (
          <div className="mt-4 space-y-1">
            {report.findings.map((f, i) => (
              <p key={i} className="text-xs text-amber-300">
                {String(f.severity || "info")}: {String(f.message || JSON.stringify(f))}
              </p>
            ))}
          </div>
        )}
      </TabsContent>

      <TabsContent value="reputation">
        <dl className="grid grid-cols-2 gap-3 text-sm md:grid-cols-4">
          <KV k="Type" v={rep?.type} />
          <KV k="Risk" v={rep?.risk != null ? String(rep.risk) : undefined} />
          <KV k="Confidence" v={rep?.confidence != null ? `${rep.confidence}%` : undefined} />
          <KV k="Provider" v={rep?.provider} />
          <KV k="Range" v={rep?.range} mono />
          <KV k="City" v={rep?.city} />
          <KV k="Region" v={rep?.region} />
          <KV
            k="Country"
            v={
              rep?.country || rep?.country_code ? (
                <CountryLabel code={rep.country_code} name={rep.country} />
              ) : undefined
            }
          />
          <KV k="Devices in subnet" v={rep?.devices_in_subnet != null ? String(rep.devices_in_subnet) : undefined} />
        </dl>
        <div className="mt-4 flex flex-wrap gap-1.5">
          {(rep?.flags || []).map((f) => (
            <Badge key={f} className="border-primary/30 bg-primary/10 text-primary">
              {f}
            </Badge>
          ))}
          {flagPill("residential", rep?.residential)}
          {flagPill("hosting", rep?.hosting)}
          {flagPill("vpn", rep?.vpn)}
          {flagPill("proxy", rep?.proxy)}
          {flagPill("tor", rep?.tor)}
        </div>
      </TabsContent>

      <TabsContent value="consensus">
        <div className="grid gap-4 md:grid-cols-2">
          <ConsensusTable title="IPv4" rows={report.consensus?.ipv4} />
          <ConsensusTable title="IPv6" rows={report.consensus?.ipv6} />
        </div>
      </TabsContent>

      <TabsContent value="geo">
        <GeoGroup title="CDN edges" items={report.geo?.cdn} />
        <GeoGroup title="GeoIP providers" items={report.geo?.geoip} />
        <GeoGroup title="Services" items={report.geo?.services} />
      </TabsContent>

      <TabsContent value="portal">
        <div className="mb-3 flex flex-wrap gap-2 text-xs">
          <Badge className={statusToneClass("ok")}>{report.connectivity_checks?.ok ?? 0} ok</Badge>
          <Badge className={statusToneClass("warn")}>
            {report.connectivity_checks?.captive_portal ?? 0} captive
          </Badge>
          <Badge className={statusToneClass("warn")}>
            {report.connectivity_checks?.altered ?? 0} altered
          </Badge>
          <Badge className={statusToneClass("err")}>
            {report.connectivity_checks?.unreachable ?? 0} unreachable
          </Badge>
        </div>
        <table className="w-full text-left text-sm">
          <thead className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="py-1 font-medium">Endpoint</th>
              <th className="py-1 font-medium">Verdict</th>
              <th className="py-1 font-medium">Status</th>
              <th className="py-1 font-medium">RTT</th>
            </tr>
          </thead>
          <tbody>
            {(report.connectivity_checks?.endpoints || []).map((e) => (
              <tr key={e.id} className="border-t border-border/60">
                <td className="py-1.5">{e.name}</td>
                <td className="py-1.5">
                  <StatePill value={e.verdict} />
                </td>
                <td className="py-1.5 font-mono text-xs">
                  {e.status}/{e.expected_status}
                </td>
                <td className="py-1.5 font-mono text-xs">{fmtMs(e.rtt_ms)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TabsContent>

      <TabsContent value="stash">
        <table className="w-full text-left text-sm">
          <thead className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="py-1 font-medium">Service</th>
              <th className="py-1 font-medium">State</th>
              <th className="py-1 font-medium">Region</th>
              <th className="py-1 font-medium">RTT</th>
              <th className="py-1 font-medium">Detail</th>
            </tr>
          </thead>
          <tbody>
            {(report.stash_checks || []).map((s) => (
              <tr key={s.id} className="border-t border-border/60">
                <td className="py-1.5">{s.name}</td>
                <td className="py-1.5">
                  <StatePill value={s.state} />
                </td>
                <td className="py-1.5 font-mono text-xs">{s.region || "—"}</td>
                <td className="py-1.5 font-mono text-xs">{fmtMs(s.rtt_ms)}</td>
                <td className="py-1.5 text-xs text-muted-foreground">{s.detail || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TabsContent>

      <TabsContent value="ai">
        <table className="w-full text-left text-sm">
          <thead className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="py-1 font-medium">Endpoint</th>
              <th className="py-1 font-medium">State</th>
              <th className="py-1 font-medium">HTTP</th>
              <th className="py-1 font-medium">RTT</th>
              <th className="py-1 font-medium">Detail</th>
            </tr>
          </thead>
          <tbody>
            {(report.ai_endpoints || []).map((a) => (
              <tr key={a.id} className="border-t border-border/60">
                <td className="py-1.5">{a.name}</td>
                <td className="py-1.5">
                  <StatePill value={a.state} />
                </td>
                <td className="py-1.5 font-mono text-xs">{a.http_status ?? "—"}</td>
                <td className="py-1.5 font-mono text-xs">{fmtMs(a.rtt_ms)}</td>
                <td className="py-1.5 text-xs text-muted-foreground">{a.detail || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TabsContent>

      <TabsContent value="path">
        {!report.connectivity?.targets?.length ? (
          <p className="text-sm text-muted-foreground">
            Path traces are often empty when geocheck runs through SOCKS5 (ICMP cannot be
            proxied). HTTP sections above are the source of truth.
          </p>
        ) : (
          <>
            <dl className="mb-4 grid grid-cols-2 gap-3 text-sm md:grid-cols-4">
              <KV k="Score" v={report.connectivity.score != null ? String(report.connectivity.score) : undefined} />
              <KV k="Floor" v={fmtMs(report.connectivity.latency_floor_ms)} />
              <KV k="ICMP" v={report.connectivity.icmp_available ? "yes" : "no"} />
              <KV k="Privileged" v={report.connectivity.privileged ? "yes" : "no"} />
            </dl>
            <table className="w-full text-left text-sm">
              <thead className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                <tr>
                  <th className="py-1 font-medium">Target</th>
                  <th className="py-1 font-medium">Verdict</th>
                  <th className="py-1 font-medium">Score</th>
                  <th className="py-1 font-medium">RTT</th>
                  <th className="py-1 font-medium">Excess</th>
                </tr>
              </thead>
              <tbody>
                {report.connectivity.targets.map((t) => (
                  <tr key={t.id} className="border-t border-border/60">
                    <td className="py-1.5">{t.name}</td>
                    <td className="py-1.5">
                      <StatePill value={t.verdict} />
                    </td>
                    <td className="py-1.5 font-mono text-xs">{t.score ?? "—"}</td>
                    <td className="py-1.5 font-mono text-xs">{fmtMs(t.rtt_ms)}</td>
                    <td className="py-1.5 font-mono text-xs">{fmtMs(t.excess_ms)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </TabsContent>
    </Tabs>
  );
}

function KV({
  k,
  v,
  mono,
  sensitive,
}: {
  k: string;
  v?: ReactNode;
  mono?: boolean;
  sensitive?: boolean;
}) {
  const { redact } = usePrivacy();
  const hidden = sensitive && redact;
  return (
    <div>
      <dt className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">{k}</dt>
      <dd className={`mt-0.5 break-all ${mono ? "font-mono text-xs" : "text-sm"}`}>
        {hidden ? <span className="font-mono tracking-widest text-muted-foreground">••••</span> : v || "—"}
      </dd>
    </div>
  );
}

function StatePill({ value }: { value?: string }) {
  if (!value) {
    return <span className="font-mono text-xs text-muted-foreground">—</span>;
  }
  return <Badge className={statusToneClass(statusTone(value))}>{value}</Badge>;
}

function flagPill(name: string, on?: boolean) {
  if (!on) return null;
  return (
    <Badge key={name} className="border-border bg-secondary text-foreground">
      {name}
    </Badge>
  );
}

function ConsensusTable({
  title,
  rows,
}: {
  title: string;
  rows?: Array<{ code: string; country: string; count: number; total: number; percent: number }>;
}) {
  return (
    <div>
      <h4 className="mb-2 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">{title}</h4>
      {!rows?.length ? (
        <p className="text-sm text-muted-foreground">No data</p>
      ) : (
        <ul className="space-y-2">
          {rows.map((r) => (
            <li key={r.code}>
              <div className="mb-1 flex justify-between text-xs">
                <span className="inline-flex items-center gap-1.5">
                  <CountryLabel code={r.code} name={r.country} showCode />
                </span>
                <span className="font-mono">
                  {r.percent}% · {r.count}/{r.total}
                </span>
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-muted">
                <div className="h-full bg-primary" style={{ width: `${Math.min(100, r.percent)}%` }} />
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function GeoGroup({ title, items }: { title: string; items?: GeoEndpoint[] }) {
  if (!items?.length) return null;
  return (
    <div className="mb-6">
      <h4 className="mb-2 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">{title}</h4>
      <div className="grid gap-1">
        {items.map((item) => (
          <div
            key={item.id}
            className="grid grid-cols-[1.2fr_1fr_1fr] gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-secondary/40"
          >
            <div>
              {item.name}
              {item.kind && (
                <span className="ml-2 font-mono text-[10px] text-muted-foreground">{item.kind}</span>
              )}
            </div>
            <ValueCell label="v4" v={item.ipv4} />
            <ValueCell label="v6" v={item.ipv6} />
          </div>
        ))}
      </div>
    </div>
  );
}

function ValueCell({ label, v }: { label: string; v?: GeoValue }) {
  if (!v || (!v.value && !v.error && !v.country)) {
    return <span className="font-mono text-xs text-muted-foreground">{label} —</span>;
  }
  if (v.error) {
    return (
      <span className="truncate font-mono text-[11px] text-destructive" title={v.error}>
        {label} err
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 font-mono text-xs">
      {label} {v.value}
      {v.country && v.value && v.value.length === 2 ? <Flag code={v.value} /> : null}
    </span>
  );
}
