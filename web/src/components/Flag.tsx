import type { ComponentType } from "react";
import * as CountryFlags from "country-flag-icons/react/3x2";
import { cn } from "@/lib/utils";

const FLAGS = CountryFlags as Record<string, ComponentType<{ className?: string; title?: string }>>;

const RI_A = 0x1f1e6;
const RI_Z = 0x1f1ff;

export function Flag({
  code,
  className,
  title,
}: {
  code?: string | null;
  className?: string;
  title?: string;
}) {
  const cc = (code || "").trim().toUpperCase();
  if (cc.length !== 2) return null;
  const Comp = FLAGS[cc];
  if (!Comp) return null;
  return (
    <Comp
      title={title || cc}
      className={cn("inline-block h-3 w-[1.125rem] shrink-0 rounded-[2px] align-middle ring-1 ring-white/10", className)}
    />
  );
}

type FlagPart = { cc: string } | { text: string };

export function splitFlagText(text: string): FlagPart[] {
  const chars = [...text];
  const out: FlagPart[] = [];
  let buf = "";
  const flush = () => {
    if (!buf) return;
    out.push({ text: buf });
    buf = "";
  };
  for (let i = 0; i < chars.length; i++) {
    const a = chars[i].codePointAt(0);
    const b = chars[i + 1]?.codePointAt(0);
    if (a != null && b != null && a >= RI_A && a <= RI_Z && b >= RI_A && b <= RI_Z) {
      flush();
      out.push({
        cc: String.fromCharCode(a - RI_A + 65, b - RI_A + 65),
      });
      i++;
      continue;
    }
    buf += chars[i];
  }
  flush();
  return out;
}

export function FlagText({
  text,
  className,
}: {
  text: string;
  className?: string;
}) {
  const parts = splitFlagText(text);
  if (parts.length === 1 && "text" in parts[0]) {
    return <span className={className}>{parts[0].text}</span>;
  }
  return (
    <span className={cn("inline-flex min-w-0 items-center gap-1.5", className)}>
      {parts.map((part, i) =>
        "cc" in part ? (
          <Flag key={`${part.cc}-${i}`} code={part.cc} />
        ) : (
          <span key={i} className="min-w-0 truncate">
            {part.text}
          </span>
        ),
      )}
    </span>
  );
}

export function CountryLabel({
  code,
  name,
  showCode = false,
}: {
  code?: string;
  name?: string;
  showCode?: boolean;
}) {
  if (!code && !name) return null;
  return (
    <span className="inline-flex items-center gap-1.5">
      <Flag code={code} />
      <span>
        {name || code}
        {showCode && name && code ? ` (${code})` : ""}
      </span>
    </span>
  );
}
