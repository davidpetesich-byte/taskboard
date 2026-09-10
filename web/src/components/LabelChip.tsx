import { X } from "lucide-react";
import type { Label } from "../api/client";

type LabelChipVariant = "default" | "board";

export default function LabelChip({
  label,
  onRemove,
  onClick,
  active,
  disabled,
  variant = "default",
}: {
  label: Label;
  onRemove?: () => void;
  onClick?: () => void;
  active?: boolean;
  disabled?: boolean;
  variant?: LabelChipVariant;
}) {
  const color = label.color || "#6b7280";
  const isBoard = variant === "board";
  const className = isBoard
    ? `inline-flex items-center gap-1 rounded-[3px] border bg-white px-1.5 py-0.5 text-[11px] font-medium text-slate-700 ${
        active
          ? "border-blue-500 ring-1 ring-blue-500"
          : "border-slate-300"
      }`
    : `inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium text-slate-100 ${
        active ? "ring-1 ring-white/70" : ""
      }`;
  const style = isBoard
    ? undefined
    : { backgroundColor: color + "1a", borderColor: color };

  return (
    <span
      className={`${className} ${disabled ? "opacity-50" : ""}`}
      style={style}
    >
      {isBoard && (
        <span
          aria-hidden="true"
          className="h-2.5 w-2.5 shrink-0 rounded-[2px]"
          style={{ backgroundColor: color }}
        />
      )}
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
          className="hover:opacity-70 disabled:cursor-not-allowed"
        >
          <X className="w-3 h-3" />
        </button>
      )}
    </span>
  );
}
