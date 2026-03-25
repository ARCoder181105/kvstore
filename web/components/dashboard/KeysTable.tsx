"use client";

import { useState } from "react";
import { useKeys, useDeleteKey } from "@/hooks/useKeys";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Trash2, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";

export function KeysTable() {
  const [search, setSearch] = useState("");
  // Determine if search contains wildcards, if not, wrapping in * to make it a fuzzy search
  const pattern = search ? (search.includes("*") ? search : `*${search}*`) : "*";
  const { data, isLoading } = useKeys(pattern);
  const deleteKey = useDeleteKey();

  return (
    <div className="space-y-4">
      <div className="relative">
        <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-zinc-500" />
        <Input
          placeholder="Search keys... (use * for wildcards)"
          className="pl-9 bg-zinc-900 border-zinc-800 text-zinc-100 font-mono"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="rounded-md border border-zinc-800 bg-zinc-900 overflow-x-auto relative">
        <Table className="min-w-[500px]">
          <TableHeader className="bg-zinc-950">
            <TableRow className="border-zinc-800 hover:bg-transparent">
              <TableHead className="w-[30%] text-zinc-400 whitespace-nowrap">Key</TableHead>
              <TableHead className="w-[45%] text-zinc-400">Value</TableHead>
              <TableHead className="w-[15%] text-zinc-400 whitespace-nowrap">TTL</TableHead>
              <TableHead className="w-[10%] text-right text-zinc-400 whitespace-nowrap">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow className="border-zinc-800 hover:bg-zinc-800/50 transition-colors">
                <TableCell colSpan={4} className="h-24 text-center text-zinc-500">
                  Loading keys...
                </TableCell>
              </TableRow>
            ) : !data?.keys || data.keys.length === 0 ? (
              <TableRow className="border-zinc-800 hover:bg-zinc-800/50 transition-colors">
                <TableCell colSpan={4} className="h-24 text-center text-zinc-500">
                  No keys found.
                </TableCell>
              </TableRow>
            ) : (
              data.keys.map((entry) => (
                <TableRow key={entry.key} className="border-zinc-800 hover:bg-zinc-800/50 transition-colors">
                  <TableCell className="font-mono font-medium text-zinc-200">
                    {entry.key}
                  </TableCell>
                  <TableCell className="font-mono text-zinc-400 truncate max-w-[200px]" title={entry.value}>
                    {entry.value}
                  </TableCell>
                  <TableCell>
                    {entry.ttl === -1 ? (
                      <Badge variant="outline" className="border-zinc-700 text-zinc-400 font-mono">
                        forever
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="bg-zinc-800 text-zinc-300 font-mono hover:bg-zinc-800">
                        {entry.ttl}s
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-zinc-500 hover:text-red-400 hover:bg-red-400/10"
                      onClick={() => deleteKey.mutate(entry.key)}
                      disabled={deleteKey.isPending}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
