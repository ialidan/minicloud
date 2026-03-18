import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { FilesResponse } from "@/lib/types";

export function useFiles(path: string, search: string, category?: string) {
  const params = new URLSearchParams();
  if (path) params.set("path", path);
  if (search) params.set("q", search);
  if (category && category !== "all") params.set("category", category);
  const qs = params.toString();

  return useQuery({
    queryKey: ["files", path, search, category ?? "all"],
    queryFn: () => api.get<FilesResponse>(`/files${qs ? `?${qs}` : ""}`),
  });
}
