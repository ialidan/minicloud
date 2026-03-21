import { Files, Image, FileText } from "lucide-react";
import type { FileCategory } from "@/lib/types";

export const FILE_CATEGORIES: { value: FileCategory; label: string; icon: typeof Files }[] = [
  { value: "all", label: "All Files", icon: Files },
  { value: "media", label: "Media", icon: Image },
  { value: "documents", label: "Documents", icon: FileText },
];
