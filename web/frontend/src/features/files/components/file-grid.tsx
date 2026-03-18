import { Folder, FolderOpen, Trash2 } from "lucide-react";
import type { FileItem, Directory } from "@/lib/types";
import { formatBytes, isImageMime, isVideoMime } from "@/lib/format";
import { fileUrl } from "@/lib/api";
import { FileIcon } from "./file-icon";
import { FileRowActions } from "./file-row-actions";

interface FileGridProps {
  files: FileItem[];
  directories: Directory[];
  currentPath: string;
  onNavigateDir: (path: string) => void;
  onDeleteFile: (file: FileItem) => void;
  onMoveFile: (file: FileItem) => void;
  onDeleteDir: (dir: Directory) => void;
  onPreviewFile?: (file: FileItem) => void;
  showPath?: boolean;
}

export function FileGrid({
  files,
  directories,
  currentPath,
  onNavigateDir,
  onDeleteFile,
  onMoveFile,
  onDeleteDir,
  onPreviewFile,
  showPath,
}: FileGridProps) {
  return (
    <div className="space-y-3">
      {directories.length > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
          {directories.map((dir) => {
            const dirPath = currentPath
              ? `${currentPath}/${dir.name}`
              : `/${dir.name}`;
            return (
              <div
                key={`dir-${dir.id}`}
                className="group relative rounded-lg border border-border bg-surface overflow-hidden hover:border-primary/30 transition-colors"
              >
                <button
                  onClick={() => onNavigateDir(dirPath)}
                  className="flex w-full items-center gap-2 px-3 py-2.5 cursor-pointer"
                >
                  <Folder className="h-5 w-5 text-primary shrink-0" />
                  <span className="text-sm font-medium truncate">
                    {dir.name}
                  </span>
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onDeleteDir(dir);
                  }}
                  className="absolute right-1.5 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-danger/10 text-muted-foreground hover:text-danger cursor-pointer"
                  title="Delete folder"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            );
          })}
        </div>
      )}
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
      {files.map((file) => (
        <div
          key={`file-${file.id}`}
          className="group relative rounded-lg border border-border bg-surface overflow-hidden hover:border-primary/30 transition-colors"
        >
          <button
            onClick={() => onPreviewFile?.(file)}
            className="w-full cursor-pointer"
          >
            {isImageMime(file.mime_type) ? (
              <img
                src={fileUrl(file.id)}
                alt={file.original_name}
                loading="lazy"
                className="aspect-square w-full object-cover"
              />
            ) : isVideoMime(file.mime_type) ? (
              <video
                src={fileUrl(file.id)}
                preload="metadata"
                muted
                className="aspect-square w-full object-cover"
              />
            ) : (
              <div className="flex aspect-square w-full items-center justify-center bg-muted">
                <FileIcon mimeType={file.mime_type} size="lg" />
              </div>
            )}
          </button>
          <div className="p-2">
            <p className="text-xs font-medium truncate">{file.original_name}</p>
            <p className="text-xs text-muted-foreground">{formatBytes(file.size)}</p>
            {showPath && (
              <p className="mt-0.5 flex items-center gap-1 text-[10px] text-muted-foreground truncate">
                <FolderOpen className="h-2.5 w-2.5 shrink-0" />
                {file.virtual_path || "/"}
              </p>
            )}
          </div>
          <div className="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <FileRowActions
              file={file}
              onDelete={onDeleteFile}
              onMove={onMoveFile}
            />
          </div>
        </div>
      ))}
      </div>
    </div>
  );
}
