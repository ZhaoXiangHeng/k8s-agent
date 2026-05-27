import type { ReactNode } from "react";

export function Notice({ type = "info", children }: { type?: "info" | "error"; children: ReactNode }) {
  return <div className={`notice ${type}`}>{children}</div>;
}
