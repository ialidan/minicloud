import type { FileCategory } from "@/lib/types";
import { cn } from "@/lib/cn";
import { FILE_CATEGORIES } from "@/lib/constants";

interface CategoryTabsProps {
  active: FileCategory;
  onChange: (category: FileCategory) => void;
}

export function CategoryTabs({ active, onChange }: CategoryTabsProps) {
  return (
    <div role="tablist" className="flex border-b border-border">
      {FILE_CATEGORIES.map(({ value, label, icon: Icon }) => (
        <button
          key={value}
          role="tab"
          aria-selected={active === value}
          onClick={() => onChange(value)}
          className={cn(
            "flex items-center gap-1.5 px-4 py-2 text-sm transition-colors cursor-pointer border-b-2 -mb-px",
            active === value
              ? "border-primary text-primary font-medium"
              : "border-transparent text-muted-foreground hover:text-foreground",
          )}
        >
          <Icon className="h-4 w-4" />
          {label}
        </button>
      ))}
    </div>
  );
}
