import { useRef, useState } from "react";
import { Plus } from "lucide-react";
import type { Label } from "../api/client";
import LabelChip from "./LabelChip";

const LABEL_COLORS = ["#EF4444", "#F59E0B", "#10B981", "#3B82F6", "#8B5CF6", "#EC4899", "#6B7280"];

export default function LabelPicker({
  allLabels,
  selectedIds,
  onChange,
  onCreate,
}: {
  allLabels: Label[];
  selectedIds: string[];
  onChange: (ids: string[]) => void;
  onCreate: (name: string, color: string) => Promise<Label>;
}) {
  const [open, setOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [newColor, setNewColor] = useState(LABEL_COLORS[0]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const busyRef = useRef(false);
  const selectedIdsRef = useRef(selectedIds);
  selectedIdsRef.current = selectedIds;

  const selected = allLabels.filter((l) => selectedIds.includes(l.id));
  const available = allLabels.filter((l) => !selectedIds.includes(l.id));

  const add = (id: string) => {
    if (busyRef.current) return;
    onChange([...selectedIds, id]);
  };
  const remove = (id: string) => {
    if (busyRef.current) return;
    onChange(selectedIds.filter((x) => x !== id));
  };

  // No <form> here: this component is mounted inside CreateTicketModal's form,
  // and a nested form would make Enter submit the ticket instead of the label.
  const handleCreate = async () => {
    const name = newName.trim();
    if (!name || busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError("");
    try {
      const created = await onCreate(name, newColor);
      const currentIds = selectedIdsRef.current;
      onChange(currentIds.includes(created.id) ? currentIds : [...currentIds, created.id]);
      setNewName("");
    } catch {
      setError("Could not create label. Please try again.");
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-1.5">
        {selected.map((l) => (
          <LabelChip key={l.id} label={l} onRemove={() => remove(l.id)} disabled={busy} />
        ))}
        <button
          type="button"
          onClick={() => {
            if (!busyRef.current) setOpen((o) => !o);
          }}
          disabled={busy}
          className="inline-flex items-center gap-1 rounded-full border border-dashed border-slate-600 px-2 py-0.5 text-[11px] text-slate-400 hover:text-slate-200 hover:border-slate-400 disabled:cursor-not-allowed disabled:opacity-50 transition-colors"
        >
          <Plus className="w-3 h-3" />
          Add label
        </button>
      </div>

      {open && (
        <div className="rounded-lg border border-slate-700 bg-slate-800/60 p-3 space-y-3">
          {available.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {available.map((l) => (
                <LabelChip key={l.id} label={l} onClick={() => add(l.id)} disabled={busy} />
              ))}
            </div>
          ) : (
            <p className="text-xs text-slate-500">
              {allLabels.length === 0 ? "No labels yet - create one below." : "All labels attached."}
            </p>
          )}

          <div className="flex flex-wrap items-center gap-2">
            <input
              value={newName}
              disabled={busy}
              onChange={(e) => {
                if (!busyRef.current) setNewName(e.target.value);
              }}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  handleCreate();
                }
              }}
              placeholder="New label…"
              className="min-w-40 flex-1 bg-slate-800 border border-slate-700 rounded-lg px-2.5 py-1 text-xs text-white placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
            />
            <div className="flex flex-wrap gap-1">
              {LABEL_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => {
                    if (!busyRef.current) setNewColor(c);
                  }}
                  disabled={busy}
                  aria-label={`Color ${c}`}
                  aria-pressed={newColor === c}
                  className={`w-6 h-6 rounded-full disabled:cursor-not-allowed disabled:opacity-50 ${newColor === c ? "ring-2 ring-white/70" : ""}`}
                  style={{ backgroundColor: c }}
                />
              ))}
            </div>
            <button
              type="button"
              onClick={handleCreate}
              disabled={busy || !newName.trim()}
              className="px-2.5 py-1 text-xs font-medium bg-slate-700 hover:bg-slate-600 disabled:opacity-40 text-slate-200 rounded-lg transition-colors"
            >
              Create
            </button>
          </div>
          {error && (
            <p role="alert" className="text-xs text-red-400">
              {error}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
