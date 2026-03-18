import { Search, FolderPlus, Upload, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ViewModeToggle } from "./view-mode-toggle";
import type { ViewMode } from "@/lib/types";

interface ToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  onUpload?: () => void;
  onNewFolder?: () => void;
  onFindDuplicates?: () => void;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  hideViewToggle?: boolean;
}

export function Toolbar({
  search,
  onSearchChange,
  onUpload,
  onNewFolder,
  onFindDuplicates,
  viewMode,
  onViewModeChange,
  hideViewToggle,
}: ToolbarProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground pointer-events-none" />
          <input
            type="text"
            placeholder="Search files..."
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            aria-label="Search files"
            className="h-9 w-full rounded-lg border border-border bg-surface pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground outline-none transition-colors focus:border-ring focus:ring-1 focus:ring-ring sm:w-64"
          />
        </div>
        {!hideViewToggle && (
          <ViewModeToggle mode={viewMode} onChange={onViewModeChange} />
        )}
      </div>
      <div className="flex items-center gap-2">
        <Button variant="secondary" size="sm" onClick={onFindDuplicates}>
          <Copy className="h-4 w-4" />
          Duplicates
        </Button>
        <Button variant="secondary" size="sm" onClick={onNewFolder}>
          <FolderPlus className="h-4 w-4" />
          New Folder
        </Button>
        <Button size="sm" onClick={onUpload}>
          <Upload className="h-4 w-4" />
          Upload
        </Button>
      </div>
    </div>
  );
}
