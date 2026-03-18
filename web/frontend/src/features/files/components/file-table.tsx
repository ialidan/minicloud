import { Folder, FolderOpen } from "lucide-react";
import type { FileItem, Directory } from "@/lib/types";
import { formatBytes, formatDate, isImageMime, isVideoMime } from "@/lib/format";
import { fileUrl } from "@/lib/api";
import { FileIcon } from "./file-icon";
import { FileRowActions, DirRowActions } from "./file-row-actions";

interface FileTableProps {
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

function displayPath(virtualPath: string): string {
  return virtualPath || "/";
}

export function FileTable({
  files,
  directories,
  currentPath,
  onNavigateDir,
  onDeleteFile,
  onMoveFile,
  onDeleteDir,
  onPreviewFile,
  showPath,
}: FileTableProps) {
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-surface">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-left text-xs text-muted-foreground">
            <th scope="col" className="px-4 py-3 font-medium">
              Name
            </th>
            {showPath && (
              <th scope="col" className="hidden px-4 py-3 font-medium sm:table-cell">
                Location
              </th>
            )}
            <th scope="col" className="hidden px-4 py-3 font-medium sm:table-cell">
              Size
            </th>
            <th scope="col" className="hidden px-4 py-3 font-medium md:table-cell">
              Modified
            </th>
            <th scope="col" className="px-4 py-3 text-right font-medium w-16">
              <span className="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {directories.map((dir) => {
            const dirPath = currentPath
              ? `${currentPath}/${dir.name}`
              : `/${dir.name}`;
            return (
              <tr
                key={`dir-${dir.id}`}
                className="border-b border-border last:border-0 hover:bg-surface-hover transition-colors"
              >
                <td className="px-4 py-3">
                  <button
                    onClick={() => onNavigateDir(dirPath)}
                    className="flex items-center gap-2 font-medium text-foreground hover:text-primary transition-colors cursor-pointer"
                  >
                    <Folder className="h-4 w-4 text-primary" />
                    {dir.name}
                  </button>
                </td>
                {showPath && (
                  <td className="hidden px-4 py-3 sm:table-cell">
                    <span className="inline-flex items-center gap-1 rounded-md bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                      <FolderOpen className="h-3 w-3" />
                      {displayPath(dir.parent_path)}
                    </span>
                  </td>
                )}
                <td className="hidden px-4 py-3 text-muted-foreground sm:table-cell">
                  &mdash;
                </td>
                <td className="hidden px-4 py-3 text-muted-foreground md:table-cell">
                  {formatDate(dir.created_at)}
                </td>
                <td className="px-4 py-3 text-right">
                  <DirRowActions dir={dir} onDelete={onDeleteDir} />
                </td>
              </tr>
            );
          })}
          {files.map((file) => (
            <tr
              key={`file-${file.id}`}
              className="border-b border-border last:border-0 hover:bg-surface-hover transition-colors"
            >
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  {isImageMime(file.mime_type) ? (
                    <img
                      src={fileUrl(file.id)}
                      alt=""
                      loading="lazy"
                      className="h-6 w-6 rounded object-cover"
                    />
                  ) : isVideoMime(file.mime_type) ? (
                    <video
                      src={fileUrl(file.id)}
                      preload="metadata"
                      muted
                      className="h-6 w-6 rounded object-cover"
                    />
                  ) : (
                    <FileIcon mimeType={file.mime_type} size="sm" />
                  )}
                  <button
                    onClick={() => onPreviewFile?.(file)}
                    className="truncate hover:text-primary hover:underline transition-colors cursor-pointer text-left"
                  >
                    {file.original_name}
                  </button>
                </div>
              </td>
              {showPath && (
                <td className="hidden px-4 py-3 sm:table-cell">
                  <span className="inline-flex items-center gap-1 rounded-md bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                    <FolderOpen className="h-3 w-3" />
                    {displayPath(file.virtual_path)}
                  </span>
                </td>
              )}
              <td className="hidden px-4 py-3 text-muted-foreground sm:table-cell">
                {formatBytes(file.size)}
              </td>
              <td className="hidden px-4 py-3 text-muted-foreground md:table-cell">
                {formatDate(file.created_at)}
              </td>
              <td className="px-4 py-3 text-right">
                <FileRowActions
                  file={file}
                  onDelete={onDeleteFile}
                  onMove={onMoveFile}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
