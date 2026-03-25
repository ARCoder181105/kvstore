"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { SetKeyPayload } from "@/lib/types";

export function useKeys(pattern?: string) {
  return useQuery({
    queryKey: ["keys", pattern],
    queryFn: () => api.getKeys(pattern),
    refetchInterval: 5000,
    staleTime: 1000,
  });
}

export function useSetKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, payload }: { key: string; payload: SetKeyPayload }) =>
      api.setKey(key, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
  });
}

export function useDeleteKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => api.deleteKey(key),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
  });
}

export function useSetExpire() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, seconds }: { key: string; seconds: number }) =>
      api.setExpire(key, seconds),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["keys"] }),
  });
}
