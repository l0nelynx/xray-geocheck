export type PingResult = {
  up: boolean;
  rttMs: number;
  status: number;
  error?: string;
  checkedAt: string;
};

export type GeoResult = {
  ok: boolean;
  error?: string;
  checkedAt: string;
  report?: GeoReport | null;
};

export type GeoValue = {
  value?: string;
  country?: string;
  error?: string;
};

export type GeoEndpoint = {
  id: string;
  name: string;
  kind?: string;
  ipv4?: GeoValue;
  ipv6?: GeoValue;
};

export type ConsensusHit = {
  code: string;
  country: string;
  count: number;
  total: number;
  percent: number;
};

export type GeoReport = {
  schema?: number;
  tool?: string;
  timestamp?: string;
  duration_ms?: number;
  identity?: {
    ipv4?: string;
    ipv6?: string;
    asn?: number;
    as_name?: string;
    org?: string;
    as_country?: string;
  };
  transport?: { resolver?: string };
  findings?: Array<{ severity?: string; message?: string; [k: string]: unknown }>;
  reputation?: {
    type?: string;
    residential?: boolean;
    risk?: number;
    confidence?: number;
    flags?: string[];
    proxy?: boolean;
    vpn?: boolean;
    tor?: boolean;
    hosting?: boolean;
    scraper?: boolean;
    compromised?: boolean;
    anonymous?: boolean;
    provider?: string;
    range?: string;
    city?: string;
    region?: string;
    country?: string;
    country_code?: string;
    devices_in_subnet?: number;
  };
  consensus?: { ipv4?: ConsensusHit[]; ipv6?: ConsensusHit[] };
  geo?: {
    cdn?: GeoEndpoint[];
    geoip?: GeoEndpoint[];
    services?: GeoEndpoint[];
  };
  connectivity_checks?: {
    clean?: boolean;
    plain_http_blocked?: boolean;
    ok?: number;
    captive_portal?: number;
    altered?: number;
    unreachable?: number;
    endpoints?: Array<{
      id: string;
      name: string;
      vendor?: string;
      url?: string;
      verdict?: string;
      status?: number;
      expected_status?: number;
      rtt_ms?: number;
    }>;
  };
  stash_checks?: Array<{
    id: string;
    name: string;
    state?: string;
    region?: string;
    detail?: string;
    rtt_ms?: number;
  }>;
  ai_endpoints?: Array<{
    id: string;
    name: string;
    vendor?: string;
    state?: string;
    http_status?: number;
    detail?: string;
    rtt_ms?: number;
  }>;
  connectivity?: {
    icmp_available?: boolean;
    privileged?: boolean;
    score?: number;
    latency_floor_ms?: number;
    breakdown?: Record<string, number>;
    targets?: Array<{
      id: string;
      name: string;
      host?: string;
      verdict?: string;
      score?: number;
      rtt_ms?: number;
      excess_ms?: number;
      loss?: number;
    }>;
  };
};

export type ProxyStatus = {
  id: string;
  remarks: string;
  address: string;
  protocol: string;
  socksAddr: string;
  ping?: PingResult;
  geo?: GeoResult;
  refreshing?: boolean;
};

export type Snapshot = {
  subscription: {
    ok: boolean;
    error?: string;
    hostCount: number;
    fetchedAt: string;
    userAgent: string;
  };
  lastPingAt?: string | null;
  lastGeoAt?: string | null;
  geoRunning: boolean;
  pingRunning: boolean;
  refreshingAll?: boolean;
  proxies: ProxyStatus[];
};
