import { useEffect, useRef, useCallback } from "react";
import { createPortal } from "react-dom";
import { X, ChevronLeft, ChevronRight, Download } from "lucide-react";
import type { FileItem } from "@/lib/types";
import { fileUrl } from "@/lib/api";
import { formatBytes, formatDate, isImageMime, isVideoMime, isAudioMime, isPdfMime, isTextMime } from "@/lib/format";
import { FileIcon } from "./file-icon";
import { TextPreview } from "./text-preview";
import { Button } from "@/components/ui/button";
import { useFocusTrap } from "@/lib/hooks/use-focus-trap";

interface FilePreviewModalProps {
  open: boolean;
  onClose: () => void;
  file: FileItem | null;
  files: FileItem[];
  onNavigate: (file: FileItem) => void;
}

export function FilePreviewModal({
  open,
  onClose,
  file,
  files,
  onNavigate,
}: FilePreviewModalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  useFocusTrap(containerRef, open);

  const currentIndex = file ? files.findIndex((f) => f.id === file.id) : -1;
  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex < files.length - 1;

  const goPrev = useCallback(() => {
    if (hasPrev) onNavigate(files[currentIndex - 1]!);
  }, [hasPrev, files, currentIndex, onNavigate]);

  const goNext = useCallback(() => {
    if (hasNext) onNavigate(files[currentIndex + 1]!);
  }, [hasNext, files, currentIndex, onNavigate]);

  useEffect(() => {
    if (!open) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
      if (e.key === "ArrowLeft") goPrev();
      if (e.key === "ArrowRight") goNext();
    }

    document.addEventListener("keydown", handleKeyDown);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = prev;
    };
  }, [open, onClose, goPrev, goNext]);

  if (!open || !file) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[100] flex flex-col bg-black/90 backdrop-blur-sm"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 text-white">
        <div className="flex items-center gap-3 min-w-0">
          <h2 className="text-sm font-medium truncate">{file.original_name}</h2>
          <span className="text-xs text-white/60 shrink-0">
            {formatBytes(file.size)}
          </span>
          {file.media?.taken_at && (
            <span className="text-xs text-white/60 shrink-0">
              {new Date(file.media.taken_at).toLocaleDateString(undefined, {
                month: "short",
                day: "numeric",
                year: "numeric",
              })}
            </span>
          )}
          {file.media?.camera_model && (
            <span className="text-xs text-white/40 shrink-0 hidden sm:inline">
              {file.media.camera_model}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => window.open(fileUrl(file.id), "_blank")}
            aria-label="Download"
            className="text-white hover:bg-white/10"
          >
            <Download className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            aria-label="Close preview"
            className="text-white hover:bg-white/10"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Content */}
      <div
        ref={containerRef}
        className="relative flex flex-1 items-center justify-center overflow-hidden px-12"
        role="dialog"
        aria-modal="true"
        aria-label={`Preview: ${file.original_name}`}
      >
        {/* Prev button */}
        {hasPrev && (
          <button
            onClick={goPrev}
            className="absolute left-2 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-2 text-white hover:bg-white/20 transition-colors cursor-pointer z-10"
            aria-label="Previous file"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
        )}

        {/* Preview content */}
        <PreviewContent file={file} />

        {/* Next button */}
        {hasNext && (
          <button
            onClick={goNext}
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-2 text-white hover:bg-white/20 transition-colors cursor-pointer z-10"
            aria-label="Next file"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
        )}
      </div>

      {/* Counter */}
      {files.length > 1 && (
        <div className="py-2 text-center text-xs text-white/50">
          {currentIndex + 1} / {files.length}
        </div>
      )}
    </div>,
    document.body,
  );
}

function PreviewContent({ file }: { file: FileItem }) {
  const mime = file.mime_type;
  const url = fileUrl(file.id);

  if (isImageMime(mime)) {
    return (
      <img
        src={url}
        alt={file.original_name}
        className="max-h-[80vh] max-w-full object-contain rounded"
      />
    );
  }

  if (isVideoMime(mime)) {
    return (
      <video
        key={file.id}
        src={url}
        controls
        className="max-h-[80vh] max-w-full rounded"
      />
    );
  }

  if (isAudioMime(mime)) {
    return (
      <div className="flex flex-col items-center gap-4">
        <FileIcon mimeType={mime} size="lg" />
        <p className="text-sm text-white">{file.original_name}</p>
        <audio key={file.id} src={url} controls className="w-full max-w-md" />
      </div>
    );
  }

  if (isPdfMime(mime)) {
    return (
      <iframe
        src={url}
        title={file.original_name}
        className="h-[80vh] w-full max-w-4xl rounded-lg bg-white"
      />
    );
  }

  if (isTextMime(mime)) {
    return <TextPreview fileId={file.id} />;
  }

  // Fallback: file info card
  return (
    <div className="rounded-xl border border-border bg-surface p-8 text-center max-w-sm">
      <FileIcon mimeType={mime} size="lg" />
      <h3 className="mt-4 text-sm font-medium text-foreground">
        {file.original_name}
      </h3>
      <p className="mt-1 text-xs text-muted-foreground">
        {formatBytes(file.size)} &middot; {mime}
      </p>
      <p className="text-xs text-muted-foreground">
        {formatDate(file.created_at)}
      </p>
      <Button
        className="mt-4"
        size="sm"
        onClick={() => window.open(url, "_blank")}
      >
        <Download className="h-4 w-4" />
        Download
      </Button>
    </div>
  );
}
