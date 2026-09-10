import { useCallback, useEffect, useRef, useState } from "react";
import { X, Trash2, CheckCircle2, Circle, Pencil, Eye } from "lucide-react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, type Ticket, type TicketInput, type Project, type Team, type Subtask, type Label } from "../api/client";
import { STATUSES, STATUS_LABELS } from "../constants/statuses";
import LabelPicker from "./LabelPicker";

const PRIORITIES = ["urgent", "high", "medium", "low"];

const mergeLabels = (current: Label[], incoming: Label[]) => {
  const seen = new Set<string>();
  return [...current, ...incoming].filter((label) => {
    if (seen.has(label.id)) return false;
    seen.add(label.id);
    return true;
  });
};

// The API returns RFC 3339 timestamps; <input type="date"> only accepts YYYY-MM-DD.
const toDateInput = (value?: string) => (value ? value.slice(0, 10) : "");

const sameSubtasks = (a: Subtask[], b: Subtask[]) =>
  a.length === b.length &&
  a.every((s, i) => s.id === b[i].id && s.title === b[i].title && s.completed === b[i].completed);

const fieldClass =
  "w-full rounded-md border border-slate-300 bg-white px-2.5 py-1.5 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500";
const iconButtonClass =
  "rounded p-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500";

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[6.5rem_minmax(0,1fr)] items-center gap-2 py-1.5">
      <span className="text-xs font-medium text-slate-600">{label}</span>
      <div className="min-w-0">{children}</div>
    </div>
  );
}

export default function TicketPanel({
  ticket,
  projects,
  teams,
  onClose,
  onUpdate,
  onDelete,
}: {
  ticket: Ticket;
  projects: Project[];
  teams: Team[];
  onClose: () => void;
  onUpdate: (id: string, data: TicketInput) => Promise<void>;
  onDelete: (id: string) => void;
}) {
  const [title, setTitle] = useState(ticket.title);
  const [description, setDescription] = useState(ticket.description || "");
  const [status, setStatus] = useState(ticket.status);
  const [priority, setPriority] = useState(ticket.priority);
  const [dueDate, setDueDate] = useState(toDateInput(ticket.dueDate));
  const [teamId, setTeamId] = useState(ticket.teamId || "");
  const [subtasks, setSubtasks] = useState<Subtask[]>(ticket.subtasks || []);
  const [newSubtask, setNewSubtask] = useState("");
  const [dirty, setDirty] = useState(false);
  const [descMode, setDescMode] = useState<"view" | "edit">(ticket.description ? "view" : "edit");
  const [labelIds, setLabelIds] = useState((ticket.labels || []).map((l) => l.id));
  const [allLabels, setAllLabels] = useState<Label[]>(() => mergeLabels([], ticket.labels || []));
  const [labelsLoading, setLabelsLoading] = useState(true);
  const [labelsError, setLabelsError] = useState("");
  const [labelBusy, setLabelBusy] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [remoteTicket, setRemoteTicket] = useState<Ticket | null>(null);
  const labelBusyRef = useRef(false);
  const savingRef = useRef(false);
  const dirtyRef = useRef(false);
  const editVersionRef = useRef(0);
  const lastSeenUpdatedAt = useRef(ticket.updatedAt);

  const project = projects.find((p) => p.id === ticket.projectId);

  useEffect(() => {
    let cancelled = false;
    api.labels
      .list()
      .then((ls) => {
        if (cancelled) return;
        setAllLabels((prev) => mergeLabels(prev, ls || []));
        setLabelsLoading(false);
      })
      .catch(() => {
        if (cancelled) return;
        setLabelsError("Could not load all labels.");
        setLabelsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // Replace every editable value with the server's version and mark the tree clean.
  const adoptTicket = useCallback((fresh: Ticket) => {
    setTitle(fresh.title);
    setDescription(fresh.description || "");
    setStatus(fresh.status);
    setPriority(fresh.priority);
    setDueDate(toDateInput(fresh.dueDate));
    setTeamId(fresh.teamId || "");
    setLabelIds((fresh.labels || []).map((l) => l.id));
    setAllLabels((prev) => mergeLabels(prev, fresh.labels || []));
    setSubtasks((prev) => (sameSubtasks(prev, fresh.subtasks || []) ? prev : fresh.subtasks || []));
    lastSeenUpdatedAt.current = fresh.updatedAt;
    editVersionRef.current += 1;
    dirtyRef.current = false;
    setDirty(false);
    setRemoteTicket(null);
    setSaveError("");
  }, []);

  const handleCreateLabel = async (name: string, color: string) => {
    labelBusyRef.current = true;
    setLabelBusy(true);
    try {
      const created = await api.labels.create({ name, color });
      setAllLabels((prev) => mergeLabels(prev, [created]));
      return created;
    } finally {
      labelBusyRef.current = false;
      setLabelBusy(false);
    }
  };

  const markDirty = () => {
    editVersionRef.current += 1;
    dirtyRef.current = true;
    setDirty(true);
    setSaveError("");
  };

  const handleSave = async () => {
    if (savingRef.current || labelBusyRef.current) return;
    const editVersion = editVersionRef.current;
    savingRef.current = true;
    setSaving(true);
    setSaveError("");
    try {
      await onUpdate(ticket.id, {
        title,
        description,
        status,
        priority,
        dueDate: dueDate || undefined,
        teamId: teamId || undefined,
        labels: labelIds,
      });
      if (editVersionRef.current === editVersion) {
        dirtyRef.current = false;
        setDirty(false);
        setRemoteTicket(null);
        try {
          const fresh = await api.tickets.get(ticket.id);
          lastSeenUpdatedAt.current = fresh.updatedAt;
          if (editVersionRef.current === editVersion) adoptTicket(fresh);
        } catch {
          // Keep local values; the next poll will reconcile.
        }
      }
    } catch {
      setSaveError("Could not save changes. Please try again.");
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  };

  const handleAddSubtask = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSubtask.trim()) return;
    const sub = await api.tickets.addSubtask(ticket.id, newSubtask);
    setSubtasks((prev) => [...prev, sub]);
    setNewSubtask("");
  };

  const handleToggleSubtask = async (id: string) => {
    const updated = await api.subtasks.toggle(id);
    setSubtasks((prev) => prev.map((s) => (s.id === id ? updated : s)));
  };

  const handleDeleteSubtask = async (id: string) => {
    await api.subtasks.delete(id);
    setSubtasks((prev) => prev.filter((s) => s.id !== id));
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6">
      <div className="absolute inset-0 bg-slate-900/50" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="ticket-modal-title"
        className="relative flex max-h-[90vh] w-[min(64rem,100vw-3rem)] flex-col overflow-hidden rounded-lg bg-white text-slate-900 shadow-[0_16px_48px_rgba(9,30,66,0.28)]"
      >
        <div className="flex shrink-0 items-center justify-between gap-3 border-b border-slate-200 px-6 py-3">
          <span className="text-xs font-medium text-slate-500">
            {ticket.projectPrefix}-{ticket.number}
          </span>
          <div className="flex items-center gap-1.5">
            {dirty && (
              <button
                type="button"
                onClick={handleSave}
                disabled={saving || labelBusy}
                className="mr-2 rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1"
              >
                {saving ? "Saving…" : "Save changes"}
              </button>
            )}
            <button
              type="button"
              aria-label="Delete ticket"
              onClick={() => {
                onDelete(ticket.id);
                onClose();
              }}
              className={`${iconButtonClass} hover:text-red-600`}
            >
              <Trash2 aria-hidden="true" className="h-4 w-4" />
            </button>
            <button type="button" aria-label="Close" onClick={onClose} className={iconButtonClass}>
              <X aria-hidden="true" className="h-5 w-5" />
            </button>
          </div>
        </div>

        {saveError && (
          <p role="alert" className="shrink-0 border-b border-red-200 bg-red-50 px-6 py-2 text-xs text-red-700">
            {saveError}
          </p>
        )}

        {remoteTicket && (
          <div className="flex shrink-0 items-center justify-between gap-3 border-b border-amber-200 bg-amber-50 px-6 py-2 text-xs text-amber-900">
            <span>This ticket changed elsewhere. Reload to see the latest version; your unsaved edits will be discarded.</span>
            <button
              type="button"
              onClick={() => adoptTicket(remoteTicket)}
              className="rounded-md border border-amber-300 bg-white px-2.5 py-1 font-medium text-amber-900 transition-colors hover:bg-amber-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            >
              Reload
            </button>
          </div>
        )}

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="grid grid-cols-1 gap-6 p-6 min-[56rem]:grid-cols-[minmax(0,1fr)_20rem]">
            <div className="min-w-0 space-y-4">
              <label htmlFor="ticket-modal-title" className="sr-only">
                Title
              </label>
              <input
                id="ticket-modal-title"
                value={title}
                onChange={(e) => {
                  setTitle(e.target.value);
                  markDirty();
                }}
                className="w-full rounded-md border border-transparent bg-transparent px-1 py-0.5 text-2xl font-semibold leading-tight text-slate-900 hover:border-slate-300 focus:border-blue-500 focus:outline-none"
              />

              <div>
                <div className="mb-1.5 flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-slate-700">Description</h3>
                  {descMode === "view" ? (
                    <button
                      type="button"
                      onClick={() => setDescMode("edit")}
                      className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                    >
                      <Pencil aria-hidden="true" className="h-3 w-3" />
                      Edit
                    </button>
                  ) : (
                    <button
                      type="button"
                      onClick={() => setDescMode("view")}
                      className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                    >
                      <Eye aria-hidden="true" className="h-3 w-3" />
                      Done
                    </button>
                  )}
                </div>
                {descMode === "edit" ? (
                  <textarea
                    value={description}
                    onChange={(e) => {
                      setDescription(e.target.value);
                      markDirty();
                    }}
                    placeholder="Add a description (supports markdown)…"
                    className="min-h-[20rem] w-full resize-y rounded-md border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                ) : description ? (
                  <div className="prose-doc rounded-md bg-slate-50 px-4 py-3">
                    <Markdown remarkPlugins={[remarkGfm]}>{description}</Markdown>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={() => setDescMode("edit")}
                    className="w-full rounded-md bg-slate-50 px-4 py-3 text-left text-sm text-slate-500 hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                  >
                    Add a description…
                  </button>
                )}
              </div>
            </div>

            <div className="space-y-4">
              <section className="rounded-md border border-slate-200">
                <h3 className="border-b border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-700">Details</h3>
                <div className="px-4 py-2">
                  <DetailRow label="Status">
                    <select
                      value={status}
                      onChange={(e) => {
                        setStatus(e.target.value);
                        markDirty();
                      }}
                      className={fieldClass}
                    >
                      {STATUSES.map((s) => (
                        <option key={s} value={s}>
                          {STATUS_LABELS[s]}
                        </option>
                      ))}
                    </select>
                  </DetailRow>
                  <DetailRow label="Priority">
                    <select
                      value={priority}
                      onChange={(e) => {
                        setPriority(e.target.value);
                        markDirty();
                      }}
                      className={`${fieldClass} capitalize`}
                    >
                      {PRIORITIES.map((p) => (
                        <option key={p} value={p}>
                          {p}
                        </option>
                      ))}
                    </select>
                  </DetailRow>
                  <DetailRow label="Due date">
                    <input
                      type="date"
                      value={dueDate}
                      onChange={(e) => {
                        setDueDate(e.target.value);
                        markDirty();
                      }}
                      className={fieldClass}
                    />
                  </DetailRow>
                  <DetailRow label="Team">
                    <select
                      value={teamId}
                      onChange={(e) => {
                        setTeamId(e.target.value);
                        markDirty();
                      }}
                      className={fieldClass}
                    >
                      <option value="">None</option>
                      {teams.map((t) => (
                        <option key={t.id} value={t.id}>
                          {t.name}
                        </option>
                      ))}
                    </select>
                  </DetailRow>
                  <DetailRow label="Project">
                    <span className="text-sm text-slate-800">{project?.name || "—"}</span>
                  </DetailRow>
                  <div className="py-1.5">
                    <span className="mb-1.5 block text-xs font-medium text-slate-600">Labels</span>
                    <LabelPicker
                      allLabels={allLabels}
                      selectedIds={labelIds}
                      onChange={(ids) => {
                        setLabelIds(ids);
                        markDirty();
                      }}
                      onCreate={handleCreateLabel}
                    />
                    {labelsLoading && <p className="mt-1.5 text-xs text-slate-500">Loading labels…</p>}
                    {labelsError && (
                      <p role="alert" className="mt-1.5 text-xs text-red-600">
                        {labelsError}
                      </p>
                    )}
                  </div>
                </div>
              </section>

              <section className="rounded-md border border-slate-200">
                <h3 className="border-b border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-700">Subtasks</h3>
                <div className="px-2 py-2">
                  <div className="space-y-0.5">
                    {subtasks.map((sub) => (
                      <div key={sub.id} className="group flex items-center gap-2.5 rounded-md px-2 py-1.5 hover:bg-slate-50">
                        <button
                          type="button"
                          aria-label={sub.completed ? `Mark ${sub.title} not done` : `Mark ${sub.title} done`}
                          onClick={() => handleToggleSubtask(sub.id)}
                          className="shrink-0"
                        >
                          {sub.completed ? (
                            <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-green-600" />
                          ) : (
                            <Circle aria-hidden="true" className="h-4 w-4 text-slate-400" />
                          )}
                        </button>
                        <span className={`flex-1 text-sm ${sub.completed ? "text-slate-500 line-through" : "text-slate-800"}`}>
                          {sub.title}
                        </span>
                        <button
                          type="button"
                          aria-label={`Delete subtask ${sub.title}`}
                          onClick={() => handleDeleteSubtask(sub.id)}
                          className="text-slate-400 opacity-0 transition-all hover:text-red-600 group-hover:opacity-100 focus-visible:opacity-100"
                        >
                          <X aria-hidden="true" className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                  <form onSubmit={handleAddSubtask} className="mt-2 flex gap-2 px-2 pb-1">
                    <input
                      value={newSubtask}
                      onChange={(e) => setNewSubtask(e.target.value)}
                      placeholder="Add subtask…"
                      className="flex-1 rounded-md border border-slate-300 bg-white px-2.5 py-1.5 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                    <button
                      type="submit"
                      className="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                    >
                      Add
                    </button>
                  </form>
                </div>
              </section>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
