import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

const KEY = "xray-geocheck.privacy";

type PrivacyState = {
  redact: boolean;
  toggle: () => void;
};

const PrivacyContext = createContext<PrivacyState>({
  redact: false,
  toggle: () => {},
});

function readStored(): boolean {
  try {
    return localStorage.getItem(KEY) === "1";
  } catch {
    return false;
  }
}

export function PrivacyProvider({ children }: { children: ReactNode }) {
  const [redact, setRedact] = useState(readStored);
  const toggle = useCallback(() => {
    setRedact((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(KEY, next ? "1" : "0");
      } catch {
        /* ignore */
      }
      return next;
    });
  }, []);
  const value = useMemo(() => ({ redact, toggle }), [redact, toggle]);
  return <PrivacyContext.Provider value={value}>{children}</PrivacyContext.Provider>;
}

export function usePrivacy() {
  return useContext(PrivacyContext);
}

export function Sensitive({ children }: { children?: ReactNode }) {
  const { redact } = usePrivacy();
  if (!redact) return <>{children}</>;
  return <span className="font-mono tracking-widest text-muted-foreground">••••</span>;
}
