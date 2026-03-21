import { Folder, FolderOpen } from "lucide-react";
import type { FileItem, Directory } from "@/lib/types";
import { formatBytes, isImageMime, isVideoMime } from "@/lib/format";
import { fileUrl } from "@/lib/api";
import { FileIcon } from "./file-icon";
import { FileRowActions, DirRowActions } from "./file-row-actions";

interface FileGalleryProps {
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

export function FileGallery({
  files,
  directories,
  currentPath,
  onNavigateDir,
  onDeleteFile,
  onMoveFile,
  onDeleteDir,
  onPreviewFile,
  showPath,
}: FileGalleryProps) {
  return (
    <div className="columns-2 sm:columns-3 md:columns-4 gap-3 space-y-3">
      {directories.map((dir) => {
        const dirPath = currentPath
          ? `${currentPath}/${dir.name}`
          : `/${dir.name}`;
        return (
          <div
            key={`dir-${dir.id}`}
            className="group relative break-inside-avoid rounded-lg border border-border bg-surface p-4 hover:border-primary/30 transition-colors"
          >
            <button
              onClick={() => onNavigateDir(dirPath)}
              className="flex w-full items-center gap-2 cursor-pointer"
            >
              <Folder className="h-5 w-5 text-primary" />
              <span className="text-sm font-medium truncate">
                {dir.name}
              </span>
            </button>
            <div className="absolute top-1 right-1 opacity-0 md:group-hover:opacity-100 transition-opacity">
              <DirRowActions dir={dir} onDelete={onDeleteDir} />
            </div>
          </div>
        );
      })}
      {files.map((file) => (
        <div
          key={`file-${file.id}`}
          className="group relative break-inside-avoid rounded-lg border border-border bg-surface overflow-hidden hover:border-primary/30 transition-colors"
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
                className="w-full object-cover"
              />
            ) : isVideoMime(file.mime_type) ? (
              <video
                src={fileUrl(file.id)}
                preload="metadata"
                muted
                className="w-full object-cover"
              />
            ) : (
              <div className="flex aspect-video w-full items-center justify-center bg-muted">
                <FileIcon mimeType={file.mime_type} size="lg" />
              </div>
            )}
          </button>
          <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent p-3 opacity-0 md:group-hover:opacity-100 transition-opacity">
            <p className="text-xs font-medium text-white truncate">
              {file.original_name}
            </p>
            <p className="text-xs text-white/70">{formatBytes(file.size)}</p>
            {showPath && (
              <p className="flex items-center gap-1 text-[10px] text-white/50 truncate">
                <FolderOpen className="h-2.5 w-2.5 shrink-0" />
                {file.virtual_path || "/"}
              </p>
            )}
          </div>
          <div className="absolute top-1 right-1 opacity-0 md:group-hover:opacity-100 transition-opacity">
            <FileRowActions
              file={file}
              onDelete={onDeleteFile}
              onMove={onMoveFile}
            />
          </div>
        </div>
      ))}
    </div>
  );
}
