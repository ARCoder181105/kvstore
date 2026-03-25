"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useStats } from "@/hooks/useStats";

function formatBytes(bytes: number): string {
  if (bytes == null) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function StatsPanel() {
  const { data: stats, isLoading } = useStats();

  const items = [
    { label: "Total Keys", value: stats?.total_keys ?? "—" },
    { label: "Memory", value: stats ? formatBytes(stats.memory_bytes) : "—" },
    { label: "TTL Keys", value: stats?.ttl_keys ?? "—" },
    { label: "Clients", value: stats?.connected_clients ?? "—" },
    { label: "Uptime", value: stats?.uptime ?? "—" },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
      {items.map((item) => (
        <Card key={item.label} className="bg-zinc-900 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-xs font-medium text-zinc-400">
              {item.label}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xl font-bold font-mono text-zinc-100">
              {item.value}
            </p>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
