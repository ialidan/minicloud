import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export function useCreateDirectory() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ path, name }: { path: string; name: string }) =>
      api.post<{ id: string }>("/directories", { path, name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
    },
  });
}
