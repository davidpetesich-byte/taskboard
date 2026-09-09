export interface StatusDef {
  key: string;
  label: string;
  /** accent dot class used in board column headers */
  dot: string;
  /** badge pill classes used in the Tickets list */
  badge: string;
}

// Single source of truth for board columns. Array order is column order.
// Mirrors BoardStatuses in internal/db/store.go - keep the two in sync.
export const STATUS_DEFS: StatusDef[] = [
  { key: "backlog",     label: "Backlog",     dot: "bg-slate-400", badge: "bg-slate-400/20 text-slate-300" },
  { key: "todo",        label: "To Do",       dot: "bg-slate-500", badge: "bg-slate-500/20 text-slate-400" },
  { key: "in_progress", label: "In Progress", dot: "bg-blue-500",  badge: "bg-blue-500/20 text-blue-400" },
  { key: "in_review",   label: "In Review",   dot: "bg-amber-500", badge: "bg-amber-500/20 text-amber-400" },
  { key: "done",        label: "Done",        dot: "bg-green-500", badge: "bg-green-500/20 text-green-400" },
];

export const STATUSES: string[] = STATUS_DEFS.map((s) => s.key);

export const STATUS_LABELS: Record<string, string> = Object.fromEntries(
  STATUS_DEFS.map((s) => [s.key, s.label]),
);

export const STATUS_COLORS: Record<string, string> = Object.fromEntries(
  STATUS_DEFS.map((s) => [s.key, s.dot]),
);

export const STATUS_STYLES: Record<string, string> = Object.fromEntries(
  STATUS_DEFS.map((s) => [s.key, s.badge]),
);
