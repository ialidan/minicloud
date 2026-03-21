import { Menu, X } from "lucide-react";
import { useState } from "react";
import type { FileCategory } from "@/lib/types";
import { cn } from "@/lib/cn";
import { FILE_CATEGORIES } from "@/lib/constants";

interface CategorySidebarProps {
  active: FileCategory;
  onChange: (category: FileCategory) => void;
}

export function CategorySidebar({ active, onChange }: CategorySidebarProps) {
  const [mobileOpen, setMobileOpen] = useState(false);

  function handleSelect(value: FileCategory) {
    onChange(value);
    setMobileOpen(false);
  }

  return (
    <>
      {/* Mobile toggle */}
      <button
        onClick={() => setMobileOpen(!mobileOpen)}
        className="flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-foreground md:hidden"
        aria-expanded={mobileOpen}
        aria-label="Toggle categories"
      >
        {mobileOpen ? (
          <X className="h-4 w-4" />
        ) : (
          <Menu className="h-4 w-4" />
        )}
        {FILE_CATEGORIES.find((i) => i.value === active)?.label}
      </button>

      {/* Sidebar */}
      <nav
        className={cn(
          "shrink-0 space-y-1",
          mobileOpen ? "block" : "hidden md:block",
        )}
      >
        {FILE_CATEGORIES.map(({ value, label, icon: Icon }) => (
          <button
            key={value}
            onClick={() => handleSelect(value)}
            className={cn(
              "flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors cursor-pointer",
              active === value
                ? "bg-primary/10 text-primary font-medium"
                : "text-muted-foreground hover:bg-surface-hover hover:text-foreground",
            )}
          >
            <Icon className="h-4 w-4 shrink-0" />
            {label}
          </button>
        ))}
      </nav>
    </>
  );
}
