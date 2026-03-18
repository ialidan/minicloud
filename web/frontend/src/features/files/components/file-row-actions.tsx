import { MoreHorizontal, Download, FolderInput, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownItem } from "@/components/ui/dropdown-menu";
import { fileUrl } from "@/lib/api";
import type { FileItem, Directory } from "@/lib/types";

interface FileRowActionsProps {
  file: FileItem;
  onDelete: (file: FileItem) => void;
  onMove: (file: FileItem) => void;
}

export function FileRowActions({
  file,
  onDelete,
  onMove,
}: FileRowActionsProps) {
  return (
    <DropdownMenu
      trigger={
        <Button variant="ghost" size="icon" aria-label="Actions">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      }
    >
      <DropdownItem
        onClick={() => window.open(fileUrl(file.id), "_blank")}
      >
        <Download className="h-4 w-4" />
        Download
      </DropdownItem>
      <DropdownItem onClick={() => onMove(file)}>
        <FolderInput className="h-4 w-4" />
        Move
      </DropdownItem>
      <DropdownItem variant="danger" onClick={() => onDelete(file)}>
        <Trash2 className="h-4 w-4" />
        Delete
      </DropdownItem>
    </DropdownMenu>
  );
}

interface DirRowActionsProps {
  dir: Directory;
  onDelete: (dir: Directory) => void;
}

export function DirRowActions({ dir, onDelete }: DirRowActionsProps) {
  return (
    <DropdownMenu
      trigger={
        <Button variant="ghost" size="icon" aria-label="Actions">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      }
    >
      <DropdownItem variant="danger" onClick={() => onDelete(dir)}>
        <Trash2 className="h-4 w-4" />
        Delete
      </DropdownItem>
    </DropdownMenu>
  );
}
