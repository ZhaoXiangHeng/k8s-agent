import type { ReactNode } from "react";

export function DataTable({ children }: { children: ReactNode }) {
  return (
    <div className="tableWrap">
      <table>{children}</table>
    </div>
  );
}
