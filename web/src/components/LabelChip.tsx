import { X } from "lucide-react";
import type { Label } from "../api/client";

export default function LabelChip({
  label,
  onRemove,
  onClick,
  active,
  disabled,
}: {
  label: Label;
  onRemove?: () => void;
  onClick?: () => void;
  active?: boolean;
  disabled?: boolean;
}) {
  const color = label.color || "#6b7280";
  const className = `inline-flex items-center gap-1 rounded-[3px] border bg-white px-1.5 py-0.5 text-[11px] font-medium text-slate-700 ${
    active ? "border-blue-500 ring-1 ring-blue-500" : "border-slate-300"
  }`;

  return (
    <span className={`${className} ${disabled ? "opacity-50" : ""}`}>
      <span
        aria-hidden="true"
        className="h-2.5 w-2.5 shrink-0 rounded-[2px]"
        style={{ backgroundColor: color }}
      />
      {onClick ? (
        <button
          type="button"
          onClick={onClick}
          disabled={disabled}
          aria-pressed={active ?? false}
          className="cursor-pointer hover:opacity-80 disabled:cursor-not-allowed"
        >
          {label.name}
        </button>
      ) : (
        label.name
      )}
      {onRemove && (
        <button
          type="button"
          disabled={disabled}
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
          aria-label={`Remove ${label.name}`}
          className="text-slate-500 hover:text-slate-800 disabled:cursor-not-allowed"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </span>
  );
}
