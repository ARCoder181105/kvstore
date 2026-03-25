"use client";

import { useState } from "react";
import { useSetKey } from "@/hooks/useKeys";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Plus } from "lucide-react";

export function AddKeyForm() {
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [ttl, setTtl] = useState("");
  const setKeyMutation = useSetKey();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!key || !value) return;

    setKeyMutation.mutate(
      {
        key,
        payload: {
          value,
          ttl: ttl ? parseInt(ttl, 10) : undefined,
        },
      },
      {
        onSuccess: () => {
          setKey("");
          setValue("");
          setTtl("");
        },
      }
    );
  };

  return (
    <Card className="bg-zinc-900 border-zinc-800">
      <CardContent className="p-4">
        <form onSubmit={handleSubmit} className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          <Input
            placeholder="Key"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            className="flex-1 bg-zinc-950 border-zinc-800 text-zinc-100 font-mono"
            required
          />
          <Input
            placeholder="Value"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            className="flex-[2] bg-zinc-950 border-zinc-800 text-zinc-100 font-mono"
            required
          />
          <Input
            placeholder="TTL (s)"
            type="number"
            min="1"
            value={ttl}
            onChange={(e) => setTtl(e.target.value)}
            className="w-24 bg-zinc-950 border-zinc-800 text-zinc-100 font-mono"
            title="Optional Time-To-Live in seconds"
          />
          <Button
            type="submit"
            disabled={setKeyMutation.isPending || !key || !value}
            className="bg-zinc-100 text-zinc-900 hover:bg-zinc-300 transition-colors"
          >
            <Plus className="mr-2 h-4 w-4" /> Add
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
