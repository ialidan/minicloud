import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { DuplicatesResponse } from "@/lib/types";

export function useDuplicates(enabled: boolean) {
  return useQuery({
    queryKey: ["duplicates"],
    queryFn: () => api.get<DuplicatesResponse>("/files/duplicates"),
    enabled,
  });
}
