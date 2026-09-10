import { useCallback, useEffect, useRef, useState } from "react";
import { X, Trash2, CheckCircle2, Circle, Pencil, Eye, MessageSquare } from "lucide-react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  api,
  type Ticket,
  type TicketInput,
  type Project,
  type Team,
  type Subtask,
  type Label,
  type Comment,
} from "../api/client";
import { STATUSES, STATUS_LABELS } from "../constants/statuses";
import LabelPicker from "./LabelPicker";

const PRIORITIES = ["urgent", "high", "medium", "low"];
const POLL_MS = 3000;

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

const sameComments = (a: Comment[], b: Comment[]) =>
  a.length === b.length &&
  a.every((c, i) => c.id === b[i].id && c.updatedAt === b[i].updatedAt && c.body === b[i].body);

const AUTHOR_KEY = "taskboard.commentAuthor";
const readAuthor = () => {
  try {
    return localStorage.getItem(AUTHOR_KEY) || "David";
  } catch {
    return "David";
  }
};

const sameSavedFields = (fresh: Ticket, saved: TicketInput) => {
  const freshLabelIds = (fresh.labels || []).map((label) => label.id);
  const savedLabelIds = saved.labels || [];
  return (
    fresh.title === saved.title &&
    (fresh.description || "") === (saved.description || "") &&
    fresh.status === saved.status &&
    fresh.priority === saved.priority &&
    toDateInput(fresh.dueDate) === (saved.dueDate || "") &&
    (fresh.teamId || "") === (saved.teamId || "") &&
    (saved.projectId === undefined || fresh.projectId === saved.projectId) &&
    freshLabelIds.length === savedLabelIds.length &&
    freshLabelIds.every((id) => savedLabelIds.includes(id))
  );
};

const fieldClass =
  "w-full rounded-md border border-slate-300 bg-white px-2.5 py-1.5 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500";
const iconButtonClass =
  "rounded p-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500";

function DetailRow({ label, htmlFor, children }: { label: string; htmlFor?: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[6.5rem_minmax(0,1fr)] items-center gap-2 py-1.5">
      {htmlFor ? (
        <label htmlFor={htmlFor} className="text-xs font-medium text-slate-600">
          {label}
        </label>
      ) : (
        <span className="text-xs font-medium text-slate-600">{label}</span>
      )}
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
  const [projectId, setProjectId] = useState(ticket.projectId);
  const [subtasks, setSubtasks] = useState<Subtask[]>(ticket.subtasks || []);
  const [newSubtask, setNewSubtask] = useState("");
  const [subtaskError, setSubtaskError] = useState("");
  const [comments, setComments] = useState<Comment[]>(ticket.comments || []);
  const [commentAuthor, setCommentAuthor] = useState(readAuthor);
  const [commentBody, setCommentBody] = useState("");
  const [commentBusy, setCommentBusy] = useState(false);
  const [commentError, setCommentError] = useState("");
  const [dirty, setDirty] = useState(false);
  const [descMode, setDescMode] = useState<"view" | "edit">("view");
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
  const pollingRef = useRef(false);
  const subtaskVersionRef = useRef(0);
  const commentVersionRef = useRef(0);
  const lastSeenUpdatedAt = useRef(ticket.updatedAt);
  // Tracks the project the server last confirmed, so the move warning clears
  // once a move is saved rather than waiting for the parent to pass a new prop.
  const savedProjectRef = useRef(ticket.projectId);

  const project = projects.find((p) => p.id === savedProjectRef.current);

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

  // Replace every editable value with the server's version and mark the tree clean.
  const adoptTicket = useCallback((fresh: Ticket, adoptSubtasks = true) => {
    setTitle(fresh.title);
    setDescription(fresh.description || "");
    setStatus(fresh.status);
    setPriority(fresh.priority);
    setDueDate(toDateInput(fresh.dueDate));
    setTeamId(fresh.teamId || "");
    setProjectId(fresh.projectId);
    savedProjectRef.current = fresh.projectId;
    setLabelIds((fresh.labels || []).map((l) => l.id));
    setAllLabels((prev) => mergeLabels(prev, fresh.labels || []));
    if (adoptSubtasks) {
      setSubtasks((prev) => (sameSubtasks(prev, fresh.subtasks || []) ? prev : fresh.subtasks || []));
      setComments((prev) => (sameComments(prev, fresh.comments || []) ? prev : fresh.comments || []));
    }
    lastSeenUpdatedAt.current = fresh.updatedAt;
    editVersionRef.current += 1;
    dirtyRef.current = false;
    setDirty(false);
    setRemoteTicket(null);
    setSaveError("");
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  useEffect(() => {
    let cancelled = false;
    const tick = async () => {
      if (cancelled || savingRef.current || pollingRef.current || document.visibilityState === "hidden") return;
      pollingRef.current = true;
      const subtaskVersion = subtaskVersionRef.current;
      const commentVersion = commentVersionRef.current;
      try {
        const fresh = await api.tickets.get(ticket.id);
        if (cancelled || savingRef.current || subtaskVersionRef.current !== subtaskVersion) return;
        setSubtasks((prev) => (sameSubtasks(prev, fresh.subtasks || []) ? prev : fresh.subtasks || []));
        const canAdoptComments = commentVersionRef.current === commentVersion;
        if (canAdoptComments) {
          setComments((prev) => (sameComments(prev, fresh.comments || []) ? prev : fresh.comments || []));
        }
        if (fresh.updatedAt === lastSeenUpdatedAt.current) return;
        if (dirtyRef.current) {
          setRemoteTicket(fresh);
        } else {
          // Subtasks are already reconciled above; the flag only has to keep an
          // in-flight comment mutation from being clobbered by this response.
          adoptTicket(fresh, canAdoptComments);
        }
      } catch {
        return; // keep the last known state; try again next tick
      } finally {
        pollingRef.current = false;
      }
    };
    // Board and list payloads carry no comments, so fetch once on open
    // instead of waiting a full interval for the first refresh.
    void tick();
    const id = window.setInterval(tick, POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [ticket.id, adoptTicket]);

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
      const savedInput: TicketInput = {
        title,
        description,
        status,
        priority,
        dueDate: dueDate || undefined,
        teamId: teamId || undefined,
        projectId,
        labels: labelIds,
      };
      await onUpdate(ticket.id, savedInput);
      if (editVersionRef.current === editVersion) {
        dirtyRef.current = false;
        setDirty(false);
        setRemoteTicket(null);
      }
      try {
        const subtaskVersion = subtaskVersionRef.current;
        const commentVersion = commentVersionRef.current;
        const fresh = await api.tickets.get(ticket.id);
        const canAdoptSide =
          subtaskVersionRef.current === subtaskVersion && commentVersionRef.current === commentVersion;
        if (editVersionRef.current === editVersion) {
          adoptTicket(fresh, canAdoptSide);
        } else {
          if (canAdoptSide) {
            setSubtasks((prev) => (sameSubtasks(prev, fresh.subtasks || []) ? prev : fresh.subtasks || []));
            setComments((prev) => (sameComments(prev, fresh.comments || []) ? prev : fresh.comments || []));
          }
          // A new local edit started while the save was settling. A response
          // matching the submitted fields is our own save and can be marked
          // seen quietly; a different response is a real remote conflict.
          if (sameSavedFields(fresh, savedInput)) {
            lastSeenUpdatedAt.current = fresh.updatedAt;
          } else {
            setRemoteTicket(fresh);
          }
        }
      } catch {
        // Keep local values; the next poll will reconcile.
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
    subtaskVersionRef.current += 1;
    setSubtaskError("");
    try {
      const sub = await api.tickets.addSubtask(ticket.id, newSubtask);
      setSubtasks((prev) => [...prev, sub]);
      setNewSubtask("");
    } catch {
      setSubtaskError("Could not add subtask. Please try again.");
    } finally {
      subtaskVersionRef.current += 1;
    }
  };

  const handleToggleSubtask = async (id: string) => {
    subtaskVersionRef.current += 1;
    setSubtaskError("");
    try {
      const updated = await api.subtasks.toggle(id);
      setSubtasks((prev) => prev.map((s) => (s.id === id ? updated : s)));
    } catch {
      setSubtaskError("Could not update subtask. Please try again.");
    } finally {
      subtaskVersionRef.current += 1;
    }
  };

  const handleDeleteSubtask = async (id: string) => {
    subtaskVersionRef.current += 1;
    setSubtaskError("");
    try {
      await api.subtasks.delete(id);
      setSubtasks((prev) => prev.filter((s) => s.id !== id));
    } catch {
      setSubtaskError("Could not delete subtask. Please try again.");
    } finally {
      subtaskVersionRef.current += 1;
    }
  };

  const handleAddComment = async () => {
    const body = commentBody.trim();
    if (!body || commentBusy) return;
    const author = commentAuthor.trim() || "David";
    try {
      localStorage.setItem(AUTHOR_KEY, author);
    } catch {
      // storage unavailable; keep going
    }
    commentVersionRef.current += 1;
    setCommentBusy(true);
    setCommentError("");
    try {
      const created = await api.tickets.addComment(ticket.id, { author, body });
      setComments((prev) => [...prev, created]);
      setCommentBody("");
    } catch {
      setCommentError("Could not add comment. Please try again.");
    } finally {
      commentVersionRef.current += 1;
      setCommentBusy(false);
    }
  };

  const handleDeleteComment = async (id: string) => {
    commentVersionRef.current += 1;
    setCommentError("");
    try {
      await api.comments.delete(id);
      setComments((prev) => prev.filter((c) => c.id !== id));
    } catch {
      setCommentError("Could not delete comment. Please try again.");
    } finally {
      commentVersionRef.current += 1;
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6">
      <div className="absolute inset-0 bg-slate-900/50" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="ticket-modal-title-label"
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
              onClick={() => adoptTicket(remoteTicket, false)}
              className="rounded-md border border-amber-300 bg-white px-2.5 py-1 font-medium text-amber-900 transition-colors hover:bg-amber-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            >
              Reload
            </button>
          </div>
        )}

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="grid grid-cols-1 gap-6 p-6 min-[56rem]:grid-cols-[minmax(0,1fr)_20rem]">
            <div className="min-w-0 space-y-4">
              <label id="ticket-modal-title-label" htmlFor="ticket-modal-title" className="sr-only">
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

              <section aria-labelledby="ticket-modal-comments-heading" className="space-y-3">
                <div className="flex items-center gap-2">
                  <MessageSquare aria-hidden="true" className="h-4 w-4 text-slate-500" />
                  <h3 id="ticket-modal-comments-heading" className="text-sm font-semibold text-slate-700">
                    Comments
                  </h3>
                  <span className="rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-600">
                    {comments.length}
                  </span>
                </div>

                {comments.length > 0 && (
                  <ul className="space-y-2">
                    {comments.map((c) => (
                      <li key={c.id} className="group rounded-md border border-slate-200 bg-white px-4 py-3">
                        <div className="mb-1.5 flex items-center gap-2">
                          <span className="text-sm font-medium text-slate-800">{c.author}</span>
                          <time dateTime={c.createdAt} className="text-xs text-slate-500">
                            {new Date(c.createdAt).toLocaleString()}
                          </time>
                          <button
                            type="button"
                            aria-label="Delete comment"
                            onClick={() => handleDeleteComment(c.id)}
                            className="ml-auto rounded p-1 text-slate-400 opacity-0 transition-all hover:bg-slate-100 hover:text-red-600 group-hover:opacity-100 focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                          >
                            <Trash2 aria-hidden="true" className="h-3.5 w-3.5" />
                          </button>
                        </div>
                        <div className="prose-doc">
                          <Markdown remarkPlugins={[remarkGfm]}>{c.body}</Markdown>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}

                <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
                  <div className="mb-2 flex items-center gap-2">
                    <label htmlFor="ticket-modal-comment-author" className="text-xs font-medium text-slate-600">
                      Comment as
                    </label>
                    <input
                      id="ticket-modal-comment-author"
                      value={commentAuthor}
                      onChange={(e) => setCommentAuthor(e.target.value)}
                      className="w-40 rounded-md border border-slate-300 bg-white px-2 py-1 text-xs text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                  <textarea
                    value={commentBody}
                    onChange={(e) => setCommentBody(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                        e.preventDefault();
                        handleAddComment();
                      }
                    }}
                    placeholder="Add a comment (supports markdown)…"
                    rows={3}
                    className="w-full resize-y rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <div className="mt-2 flex items-center gap-3">
                    <button
                      type="button"
                      onClick={handleAddComment}
                      disabled={commentBusy || !commentBody.trim()}
                      className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1"
                    >
                      {commentBusy ? "Adding…" : "Add comment"}
                    </button>
                    <span className="text-xs text-slate-500">⌘↵ to submit</span>
                    {commentError && (
                      <p role="alert" className="text-xs text-red-600">
                        {commentError}
                      </p>
                    )}
                  </div>
                </div>
              </section>
            </div>

            <div className="space-y-4">
              <section className="rounded-md border border-slate-200">
                <h3 className="border-b border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-700">Details</h3>
                <div className="px-4 py-2">
                  <DetailRow label="Status" htmlFor="ticket-modal-status">
                    <select
                      id="ticket-modal-status"
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
                  <DetailRow label="Priority" htmlFor="ticket-modal-priority">
                    <select
                      id="ticket-modal-priority"
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
                  <DetailRow label="Due date" htmlFor="ticket-modal-due-date">
                    <input
                      id="ticket-modal-due-date"
                      type="date"
                      value={dueDate}
                      onChange={(e) => {
                        setDueDate(e.target.value);
                        markDirty();
                      }}
                      className={fieldClass}
                    />
                  </DetailRow>
                  <DetailRow label="Team" htmlFor="ticket-modal-team">
                    <select
                      id="ticket-modal-team"
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
                  <DetailRow label="Project" htmlFor="ticket-modal-project">
                    <select
                      id="ticket-modal-project"
                      value={projectId}
                      onChange={(e) => {
                        setProjectId(e.target.value);
                        markDirty();
                      }}
                      className={fieldClass}
                    >
                      {projects.map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.name}
                        </option>
                      ))}
                    </select>
                    {projectId !== savedProjectRef.current && (
                      <p className="mt-1 text-xs text-amber-700">
                        Saving moves this ticket out of {project?.name || "its project"} and gives it a new
                        key in the target project.
                      </p>
                    )}
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
                  {subtaskError && (
                    <p role="alert" className="mt-1.5 px-2 text-xs text-red-600">
                      {subtaskError}
                    </p>
                  )}
                </div>
              </section>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
