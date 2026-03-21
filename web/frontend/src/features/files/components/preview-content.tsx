import { Download } from "lucide-react";
import type { FileItem } from "@/lib/types";
import { fileUrl } from "@/lib/api";
import { formatBytes, formatDate, isImageMime, isVideoMime, isAudioMime, isPdfMime, isTextMime } from "@/lib/format";
import { FileIcon } from "./file-icon";
import { TextPreview } from "./text-preview";
import { Button } from "@/components/ui/button";

export function PreviewContent({ file }: { file: FileItem }) {
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
        sandbox="allow-same-origin"
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
