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
  const className = `inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium text-slate-100 ${
    active ? "ring-1 ring-white/70" : ""
  }`;
  const style = { backgroundColor: color + "1a", borderColor: color };

  return (
    <span
      className={`${className} ${disabled ? "opacity-50" : ""}`}
      style={style}
    >
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
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          aria-label={`Remove ${label.name}`}
          className="hover:opacity-70 disabled:cursor-not-allowed"
        >
          <X className="w-3 h-3" />
        </button>
      )}
    </span>
  );
}
