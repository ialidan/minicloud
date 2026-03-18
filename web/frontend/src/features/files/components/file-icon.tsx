import { FileText, Image, Film, Music, FileCode, Archive } from "lucide-react";
import { cn } from "@/lib/cn";

const SIZE_CLASSES = {
  sm: "h-4 w-4",
  md: "h-8 w-8",
  lg: "h-12 w-12",
} as const;

interface FileIconProps {
  mimeType: string;
  size?: "sm" | "md" | "lg";
}

export function FileIcon({ mimeType, size = "sm" }: FileIconProps) {
  const sizeClass = SIZE_CLASSES[size];

  if (mimeType.startsWith("image/"))
    return <Image className={cn(sizeClass, "text-blue-500")} />;
  if (mimeType.startsWith("video/"))
    return <Film className={cn(sizeClass, "text-purple-500")} />;
  if (mimeType.startsWith("audio/"))
    return <Music className={cn(sizeClass, "text-pink-500")} />;
  if (
    mimeType.includes("zip") ||
    mimeType.includes("tar") ||
    mimeType.includes("rar") ||
    mimeType.includes("gzip")
  )
    return <Archive className={cn(sizeClass, "text-amber-500")} />;
  if (
    mimeType.includes("javascript") ||
    mimeType.includes("json") ||
    mimeType.includes("xml") ||
    mimeType.includes("html") ||
    mimeType.includes("css")
  )
    return <FileCode className={cn(sizeClass, "text-green-500")} />;
  return <FileText className={cn(sizeClass, "text-muted-foreground")} />;
}
