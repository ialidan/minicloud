import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { BASE_URL } from "@/lib/api";

interface UseUploadReturn {
  upload: (file: File, path: string, overrideName?: string) => void;
  progress: number;
  isUploading: boolean;
  error: string | null;
}

export function useUpload(
  onSuccess?: () => void,
  onError?: (msg: string) => void,
): UseUploadReturn {
  const [progress, setProgress] = useState(0);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const upload = useCallback(
    (file: File, path: string, overrideName?: string) => {
      setIsUploading(true);
      setProgress(0);
      setError(null);

      const xhr = new XMLHttpRequest();
      const formData = new FormData();

      if (overrideName) {
        formData.append("file", file, overrideName);
      } else {
        formData.append("file", file);
      }

      const params = new URLSearchParams();
      if (path) params.set("path", path);
      const qs = params.toString();

      xhr.upload.addEventListener("progress", (e) => {
        if (e.lengthComputable) {
          setProgress(Math.round((e.loaded / e.total) * 100));
        }
      });

      xhr.addEventListener("load", () => {
        setIsUploading(false);
        if (xhr.status >= 200 && xhr.status < 300) {
          queryClient.invalidateQueries({ queryKey: ["files"] });
          onSuccess?.();
        } else {
          let msg = "Upload failed";
          try {
            const body = JSON.parse(xhr.responseText) as { error?: string };
            msg = body.error ?? msg;
          } catch {
            // keep default
          }
          setError(msg);
          onError?.(msg);
        }
      });

      xhr.addEventListener("error", () => {
        setIsUploading(false);
        setError("Network error");
        onError?.("Network error");
      });

      xhr.open("POST", `${BASE_URL}/files${qs ? `?${qs}` : ""}`);
      xhr.withCredentials = true;
      xhr.send(formData);
    },
    [queryClient, onSuccess, onError],
  );

  return { upload, progress, isUploading, error };
}
