import { useRef, useState, useEffect, useMemo, memo } from "react";
import type { FileItem } from "@/lib/types";
import { groupFilesByMonth, isImageMime, isVideoMime } from "@/lib/format";
import { fileUrl } from "@/lib/api";
import { FileIcon } from "./file-icon";
import { FileRowActions } from "./file-row-actions";

interface MediaTimelineProps {
  files: FileItem[];
  onDeleteFile: (file: FileItem) => void;
  onMoveFile: (file: FileItem) => void;
  onPreviewFile?: (file: FileItem) => void;
}

export function MediaTimeline({
  files,
  onDeleteFile,
  onMoveFile,
  onPreviewFile,
}: MediaTimelineProps) {
  const groups = useMemo(() => groupFilesByMonth(files), [files]);

  return (
    <div className="space-y-8">
      {groups.map((group) => (
        <section key={group.sortKey}>
          <h3 className="sticky top-0 z-10 bg-background/95 backdrop-blur-sm py-2 text-sm font-semibold text-foreground border-b border-border mb-3">
            {group.label}
            <span className="ml-2 text-xs font-normal text-muted-foreground">
              {group.files.length}{" "}
              {group.files.length === 1 ? "item" : "items"}
            </span>
          </h3>
          <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-1.5">
            {group.files.map((file) => (
              <MediaTile
                key={file.id}
                file={file}
                onPreview={onPreviewFile}
                onDelete={onDeleteFile}
                onMove={onMoveFile}
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Individual tile
// ---------------------------------------------------------------------------

const MediaTile = memo(function MediaTile({
  file,
  onPreview,
  onDelete,
  onMove,
}: {
  file: FileItem;
  onPreview?: (file: FileItem) => void;
  onDelete: (file: FileItem) => void;
  onMove: (file: FileItem) => void;
}) {
  const isGif = file.mime_type === "image/gif";
  const isVideo = isVideoMime(file.mime_type);

  return (
    <div className="group relative aspect-square rounded-md overflow-hidden bg-muted hover:ring-2 hover:ring-primary/40 transition-all">
      <button
        onClick={() => onPreview?.(file)}
        className="w-full h-full cursor-pointer"
      >
        {isImageMime(file.mime_type) ? (
          <img
            src={fileUrl(file.id)}
            alt={file.original_name}
            loading="lazy"
            className="w-full h-full object-cover"
          />
        ) : isVideo ? (
          <VideoThumbnail file={file} />
        ) : (
          <div className="flex w-full h-full items-center justify-center">
            <FileIcon mimeType={file.mime_type} size="lg" />
          </div>
        )}
      </button>

      {/* GIF badge */}
      {isGif && (
        <span className="absolute bottom-1.5 right-1.5 bg-black/70 text-white text-[10px] font-bold px-1.5 py-0.5 rounded pointer-events-none">
          GIF
        </span>
      )}

      {/* Video duration badge (rendered inside VideoThumbnail via portal-less approach) */}
      {isVideo && <VideoDurationBadge file={file} />}

      {/* Hover overlay with actions */}
      <div className="absolute top-0.5 right-0.5 opacity-0 md:group-hover:opacity-100 transition-opacity">
        <FileRowActions file={file} onDelete={onDelete} onMove={onMove} />
      </div>
    </div>
  );
});

// ---------------------------------------------------------------------------
// Video thumbnail + duration
// ---------------------------------------------------------------------------

function VideoThumbnail({ file }: { file: FileItem }) {
  return (
    <video
      src={fileUrl(file.id)}
      preload="metadata"
      muted
      playsInline
      className="w-full h-full object-cover"
    />
  );
}

function VideoDurationBadge({ file }: { file: FileItem }) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [duration, setDuration] = useState<string | null>(null);

  useEffect(() => {
    const video = document.createElement("video");
    video.preload = "metadata";
    video.src = fileUrl(file.id);
    videoRef.current = video;

    function onLoaded() {
      if (video.duration && isFinite(video.duration)) {
        setDuration(formatDuration(video.duration));
      }
    }

    video.addEventListener("loadedmetadata", onLoaded);
    return () => {
      video.removeEventListener("loadedmetadata", onLoaded);
      video.src = "";
      videoRef.current = null;
    };
  }, [file.id]);

  if (!duration) return null;

  return (
    <span className="absolute bottom-1.5 right-1.5 bg-black/70 text-white text-[10px] font-medium px-1.5 py-0.5 rounded pointer-events-none tabular-nums">
      {duration}
    </span>
  );
}

function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);

  if (h > 0) {
    return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  }
  return `${m}:${String(s).padStart(2, "0")}`;
}
