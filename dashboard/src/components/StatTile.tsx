import type { ReactNode } from "react";

interface StatTileProps {
  label: string;
  value: string | number | null;
  icon: ReactNode;
  accent?: "default" | "green" | "red";
}

const ACCENT_CLASSES: Record<NonNullable<StatTileProps["accent"]>, string> = {
  default: "text-neutral-900 dark:text-neutral-100",
  green: "text-emerald-600 dark:text-emerald-400",
  red: "text-red-600 dark:text-red-400",
};

const ICON_BG: Record<NonNullable<StatTileProps["accent"]>, string> = {
  default: "bg-teal-500/10 text-teal-600 dark:text-teal-400",
  green: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  red: "bg-red-500/10 text-red-600 dark:text-red-400",
};

export function StatTile({
  label,
  value,
  icon,
  accent = "default",
}: StatTileProps) {
  return (
    <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm shadow-neutral-900/[0.02] dark:border-neutral-800 dark:bg-neutral-900">
      <div className="flex items-center gap-2">
        <span
          className={`flex h-7 w-7 items-center justify-center rounded-lg ${ICON_BG[accent]}`}
        >
          {icon}
        </span>
        <p className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
          {label}
        </p>
      </div>
      {value === null ? (
        <div className="mt-3 h-8 w-16 animate-pulse rounded bg-neutral-100 dark:bg-neutral-800" />
      ) : (
        <p
          className={`mt-2 text-2xl font-semibold tabular-nums ${ACCENT_CLASSES[accent]}`}
        >
          {value}
        </p>
      )}
    </div>
  );
}
