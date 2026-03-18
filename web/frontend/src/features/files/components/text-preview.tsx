import { Loader2 } from "lucide-react";
import { useFileContent } from "../hooks/use-file-content";

interface TextPreviewProps {
  fileId: string;
}

export function TextPreview({ fileId }: TextPreviewProps) {
  const { data, isLoading, error } = useFileContent(fileId, true);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
        Failed to load file content
      </div>
    );
  }

  return (
    <div className="max-h-[80vh] w-full max-w-4xl overflow-auto rounded-lg bg-surface border border-border">
      <pre className="p-4 text-sm leading-relaxed">
        <code>{data}</code>
      </pre>
    </div>
  );
}
