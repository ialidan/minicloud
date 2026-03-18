import { useQuery } from "@tanstack/react-query";
import { fileUrl } from "@/lib/api";

export function useFileContent(fileId: string | null, enabled: boolean) {
  return useQuery({
    queryKey: ["file-content", fileId],
    queryFn: async () => {
      const res = await fetch(fileUrl(fileId!), { credentials: "same-origin" });
      if (!res.ok) throw new Error("Failed to fetch file");
      return res.text();
    },
    enabled: enabled && fileId !== null,
    staleTime: 60_000,
  });
}
