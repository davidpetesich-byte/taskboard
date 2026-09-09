import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { api, type TicketInput, type Project, type Team, type Label } from "../api/client";
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
  onCreate: (data: TicketInput) => Promise<void>;
}) {
  const [projectId, setProjectId] = useState(projects[0]?.id || "");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState("medium");
  const [dueDate, setDueDate] = useState("");
  const [teamId, setTeamId] = useState("");
  const [labelIds, setLabelIds] = useState<string[]>([]);
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !projectId || labelBusyRef.current || ticketSavingRef.current) return;
    ticketSavingRef.current = true;
    setTicketSaving(true);
    setTicketError("");
    try {
      await onCreate({
        projectId,
        title,
        description,
        priority,
        status: defaultStatus || "todo",
        dueDate: dueDate || undefined,
        teamId: teamId || undefined,
        labels: labelIds,
      });
    } catch {
      setTicketError("Could not create ticket. Please try again.");
    } finally {
      ticketSavingRef.current = false;
      setTicketSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-lg bg-slate-900 border border-slate-700 rounded-xl p-6 space-y-5"
      >
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">New Ticket</h2>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-500 hover:text-slate-300 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-slate-400 mb-1.5">
              Project
            </label>
            <select
              value={projectId}
              onChange={(e) => setProjectId(e.target.value)}
              required
              className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
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
            <label className="block text-xs font-medium text-slate-400 mb-1.5">
              Title
            </label>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="What needs to be done?"
              required
              className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-400 mb-1.5">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              placeholder="Add more detail…"
              className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 resize-none"
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1.5">
                Priority
              </label>
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500 capitalize"
              >
                {PRIORITIES.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1.5">
                Due Date
              </label>
              <input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1.5">
                Team
              </label>
              <select
                value={teamId}
                onChange={(e) => setTeamId(e.target.value)}
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
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
            <label className="block text-xs font-medium text-slate-400 mb-1.5">
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
              <p role="alert" className="mt-1.5 text-xs text-red-400">
                {labelsError}
              </p>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-2">
          {ticketError && (
            <p role="alert" className="mr-auto self-center text-xs text-red-400">
              {ticketError}
            </p>
          )}
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={ticketSaving || labelBusy}
            className="px-4 py-2 text-sm font-medium bg-blue-600 hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50 text-white rounded-lg transition-colors"
          >
            {ticketSaving ? "Creating…" : "Create Ticket"}
          </button>
        </div>
      </form>
    </div>
  );
}
