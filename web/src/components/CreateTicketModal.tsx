import { useEffect, useRef, useState } from "react";
import { Circle, X } from "lucide-react";
import { api, type CreateTicketInput, type Project, type Team, type Label } from "../api/client";
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

export default function CreateTicketModal({
  projects,
  teams,
  defaultStatus,
  onClose,
  onCreate,
}: {
  projects: Project[];
  teams: Team[];
  defaultStatus?: string;
  onClose: () => void;
  onCreate: (data: CreateTicketInput) => Promise<void>;
}) {
  const [projectId, setProjectId] = useState(projects[0]?.id || "");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState("medium");
  const [dueDate, setDueDate] = useState("");
  const [teamId, setTeamId] = useState("");
  const [labelIds, setLabelIds] = useState<string[]>([]);
  const [subtasks, setSubtasks] = useState<string[]>([]);
  const [newSubtask, setNewSubtask] = useState("");
  const [allLabels, setAllLabels] = useState<Label[]>([]);
  const [labelsLoading, setLabelsLoading] = useState(true);
  const [labelsError, setLabelsError] = useState("");
  const [labelBusy, setLabelBusy] = useState(false);
  const [ticketSaving, setTicketSaving] = useState(false);
  const [ticketError, setTicketError] = useState("");
  const labelBusyRef = useRef(false);
  const ticketSavingRef = useRef(false);

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
        setLabelsError("Could not load labels.");
        setLabelsLoading(false);
      });
    return () => {
      cancelled = true;
    };
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

  const addSubtask = () => {
    const title = newSubtask.trim();
    if (!title) return;
    setSubtasks((current) => [...current, title]);
    setNewSubtask("");
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !projectId || labelBusyRef.current || ticketSavingRef.current) return;
    ticketSavingRef.current = true;
    setTicketSaving(true);
    setTicketError("");
    try {
      const pendingSubtask = newSubtask.trim();
      await onCreate({
        projectId,
        title,
        description,
        priority,
        status: defaultStatus || "todo",
        dueDate: dueDate || undefined,
        teamId: teamId || undefined,
        labels: labelIds,
        subtasks: pendingSubtask ? [...subtasks, pendingSubtask] : subtasks,
      });
    } catch {
      setTicketError("Could not create ticket. Please try again.");
    } finally {
      ticketSavingRef.current = false;
      setTicketSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-6">
      <form
        onSubmit={handleSubmit}
        className="max-h-[90vh] w-full max-w-lg space-y-5 overflow-y-auto rounded-lg bg-white p-6 text-slate-900 shadow-[0_16px_48px_rgba(9,30,66,0.28)]"
      >
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">New Ticket</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="rounded p-1 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1.5">
              Project
            </label>
            <select
              value={projectId}
              onChange={(e) => setProjectId(e.target.value)}
              required
              className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Select project…</option>
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.icon} {p.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1.5">
              Title
            </label>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="What needs to be done?"
              required
              className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1.5">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              placeholder="Add more detail…"
              className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-slate-600 mb-1.5">
                Priority
              </label>
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 capitalize"
              >
                {PRIORITIES.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-600 mb-1.5">
                Due Date
              </label>
              <input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-600 mb-1.5">
                Team
              </label>
              <select
                value={teamId}
                onChange={(e) => setTeamId(e.target.value)}
                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">None</option>
                {teams.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1.5">
              Labels
            </label>
            <LabelPicker
              allLabels={allLabels}
              selectedIds={labelIds}
              onChange={setLabelIds}
              onCreate={handleCreateLabel}
            />
            {labelsLoading && (
              <p className="mt-1.5 text-xs text-slate-500">Loading labels…</p>
            )}
            {labelsError && (
              <p role="alert" className="mt-1.5 text-xs text-red-600">
                {labelsError}
              </p>
            )}
          </div>

          <div>
            <label className="mb-1.5 block text-xs font-medium text-slate-600">
              Subtasks
            </label>
            {subtasks.length > 0 && (
              <div className="mb-2 divide-y divide-slate-200 rounded-md border border-slate-200 bg-slate-50">
                {subtasks.map((subtask, index) => (
                  <div key={`${subtask}-${index}`} className="flex items-center gap-2 px-2.5 py-2 text-sm text-slate-700">
                    <Circle className="h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
                    <span className="min-w-0 flex-1 break-words">{subtask}</span>
                    <button
                      type="button"
                      aria-label={`Remove subtask ${subtask}`}
                      onClick={() => setSubtasks((current) => current.filter((_, candidate) => candidate !== index))}
                      className="rounded p-1 text-slate-400 transition-colors hover:bg-slate-200 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
            <div className="flex gap-2">
              <input
                value={newSubtask}
                onChange={(event) => setNewSubtask(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key !== "Enter") return;
                  event.preventDefault();
                  addSubtask();
                }}
                aria-label="New subtask"
                placeholder="Add a subtask…"
                className="min-w-0 flex-1 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                type="button"
                onClick={addSubtask}
                disabled={!newSubtask.trim()}
                className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40"
              >
                Add
              </button>
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-2">
          {ticketError && (
            <p role="alert" className="mr-auto self-center text-xs text-red-600">
              {ticketError}
            </p>
          )}
          <button
            type="button"
            onClick={onClose}
            className="rounded-md px-4 py-2 text-sm text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={ticketSaving || labelBusy}
            className="px-4 py-2 text-sm font-medium bg-blue-600 hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50 text-white rounded-md transition-colors"
          >
            {ticketSaving ? "Creating…" : "Create Ticket"}
          </button>
        </div>
      </form>
    </div>
  );
}
