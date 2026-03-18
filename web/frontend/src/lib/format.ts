const UNITS = ["B", "KB", "MB", "GB", "TB"] as const;

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const value = bytes / Math.pow(k, i);
  return `${value.toFixed(i === 0 ? 0 : 1)} ${UNITS[i]}`;
}

export function formatDate(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffSec < 60) return "just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHr < 24) return `${diffHr}h ago`;
  if (diffDay < 7) return `${diffDay}d ago`;

  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: date.getFullYear() !== now.getFullYear() ? "numeric" : undefined,
  });
}

function mimeStartsWith(prefix: string) {
  return (mime: string): boolean => mime.startsWith(prefix);
}

export const isImageMime = mimeStartsWith("image/");
export const isVideoMime = mimeStartsWith("video/");
export const isAudioMime = mimeStartsWith("audio/");

export function isPdfMime(mime: string): boolean {
  return mime === "application/pdf";
}

export function isTextMime(mime: string): boolean {
  return (
    mime.startsWith("text/") ||
    mime.includes("json") ||
    mime.includes("xml") ||
    mime.includes("javascript") ||
    mime.includes("css") ||
    mime.includes("html")
  );
}

/** Groups media files by month/year based on taken_at (EXIF) or created_at fallback. */
export function groupFilesByMonth(
  files: import("@/lib/types").FileItem[],
): { label: string; sortKey: string; files: import("@/lib/types").FileItem[] }[] {
  const groups = new Map<
    string,
    { label: string; sortKey: string; files: import("@/lib/types").FileItem[] }
  >();

  for (const file of files) {
    const dateStr = file.media?.taken_at ?? file.created_at;
    const date = new Date(dateStr);
    const sortKey = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
    const label = date.toLocaleDateString(undefined, {
      month: "long",
      year: "numeric",
    });

    let group = groups.get(sortKey);
    if (!group) {
      group = { label, sortKey, files: [] };
      groups.set(sortKey, group);
    }
    group.files.push(file);
  }

  return Array.from(groups.values()).sort((a, b) =>
    b.sortKey.localeCompare(a.sortKey),
  );
}

/** Given "photo.jpg" and existing names, returns "photo (1).jpg", "photo (2).jpg", etc. */
export function getUniqueName(name: string, existingNames: string[]): string {
  const set = new Set(existingNames);
  if (!set.has(name)) return name;

  const dotIdx = name.lastIndexOf(".");
  const base = dotIdx > 0 ? name.slice(0, dotIdx) : name;
  const ext = dotIdx > 0 ? name.slice(dotIdx) : "";

  let i = 1;
  while (set.has(`${base} (${i})${ext}`)) i++;
  return `${base} (${i})${ext}`;
}
