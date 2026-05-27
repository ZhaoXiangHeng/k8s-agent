export function StatusBadge({ active, text }: { active: boolean; text: string }) {
  return <span className={`statusBadge ${active ? "ok" : "muted"}`}>{text}</span>;
}
