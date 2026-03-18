import { List, LayoutGrid, GalleryHorizontalEnd } from "lucide-react";
import type { ViewMode } from "@/lib/types";

interface ViewModeToggleProps {
  mode: ViewMode;
  onChange: (mode: ViewMode) => void;
}

const MODES: { value: ViewMode; icon: typeof List; label: string }[] = [
  { value: "table", icon: List, label: "Table view" },
  { value: "grid", icon: LayoutGrid, label: "Grid view" },
  { value: "gallery", icon: GalleryHorizontalEnd, label: "Gallery view" },
];

export function ViewModeToggle({ mode, onChange }: ViewModeToggleProps) {
  return (
    <div className="flex items-center rounded-lg border border-border bg-surface p-0.5">
      {MODES.map(({ value, icon: Icon, label }) => (
        <button
          key={value}
          onClick={() => onChange(value)}
          aria-pressed={mode === value}
          aria-label={label}
          className={`rounded-md p-1.5 transition-colors cursor-pointer ${
            mode === value
              ? "bg-surface-hover text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          <Icon className="h-4 w-4" />
        </button>
      ))}
    </div>
  );
}
