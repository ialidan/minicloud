import { FolderOpen } from "lucide-react";

interface EmptyStateProps {
  search?: string;
}

export function EmptyState({ search }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <FolderOpen className="h-12 w-12 text-muted-foreground/50" />
      {search ? (
        <>
          <p className="mt-4 text-sm font-medium text-foreground">
            No results found
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            No files or folders match &ldquo;{search}&rdquo;
          </p>
        </>
      ) : (
        <>
          <p className="mt-4 text-sm font-medium text-foreground">
            No files yet
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Upload a file or create a folder to get started
          </p>
        </>
      )}
    </div>
  );
}
