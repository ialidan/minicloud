import { useState } from "react";
import { FolderOpen, Trash2, Loader2 } from "lucide-react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { formatBytes } from "@/lib/format";
import { useDuplicates } from "../hooks/use-duplicates";
import { useDeleteFile } from "../hooks/use-file-actions";
import { useQueryClient } from "@tanstack/react-query";
import type { FileItem } from "@/lib/types";

interface FindDuplicatesModalProps {
  open: boolean;
  onClose: () => void;
}

export function FindDuplicatesModal({
  open,
  onClose,
}: FindDuplicatesModalProps) {
  const { data, isLoading } = useDuplicates(open);
  const deleteFile = useDeleteFile();
  const queryClient = useQueryClient();
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const groups = data?.duplicates ?? [];

  async function handleDelete(file: FileItem) {
    setDeletingId(file.id);
    try {
      await deleteFile.mutateAsync(file.id);
      queryClient.invalidateQueries({ queryKey: ["duplicates"] });
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Duplicate Files">
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : groups.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">
          No duplicate files found.
        </p>
      ) : (
        <div className="space-y-4 max-h-[60vh] overflow-y-auto">
          <p className="text-xs text-muted-foreground">
            {groups.length} group{groups.length !== 1 && "s"} of identical files
            found.
          </p>
          {groups.map((group) => (
            <div
              key={group.checksum}
              className="rounded-lg border border-border p-3 space-y-2"
            >
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{formatBytes(group.size)} each</span>
                <span>{group.files.length} copies</span>
              </div>
              <div className="space-y-1">
                {group.files.map((file) => (
                  <div
                    key={file.id}
                    className="flex items-center justify-between gap-2 rounded px-2 py-1.5 hover:bg-surface-hover transition-colors"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-foreground truncate">
                        {file.original_name}
                      </p>
                      <p className="flex items-center gap-1 text-xs text-muted-foreground truncate">
                        <FolderOpen className="h-3 w-3 shrink-0" />
                        {file.virtual_path || "/"}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDelete(file)}
                      disabled={deletingId === file.id}
                      aria-label={`Delete ${file.original_name}`}
                    >
                      {deletingId === file.id ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="h-4 w-4 text-muted-foreground hover:text-danger" />
                      )}
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
      <div className="flex justify-end mt-4">
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
}
