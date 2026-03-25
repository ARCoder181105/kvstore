"use client";

import { useEffect, useRef } from "react";
import { Badge } from "@/components/ui/badge";
import { useEventStream } from "@/hooks/useEventStream";
import type { StoreEvent } from "@/lib/types";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Activity } from "lucide-react";

const eventColors: Record<string, string> = {
  SET: "bg-green-500/10 text-green-400 border-green-500/20",
  DEL: "bg-red-500/10 text-red-400 border-red-500/20",
  EXPIRE: "bg-yellow-500/10 text-yellow-400 border-yellow-500/20",
  EXPIRED: "bg-zinc-500/10 text-zinc-400 border-zinc-500/20",
};

function EventRow({ event }: { event: StoreEvent }) {
  const time = new Date(event.timestamp).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
  });
  
  const colorClass = eventColors[event.type] ?? "bg-zinc-800 text-zinc-400 border-zinc-700";

  return (
    <div className="flex items-start gap-3 py-2 text-sm font-mono border-b border-zinc-800/50 last:border-0 hover:bg-zinc-800/30 px-2 rounded-md transition-colors">
      <span className="text-zinc-500 w-[85px] shrink-0 text-xs mt-0.5">{time}</span>
      <Badge variant="outline" className={`w-16 justify-center text-[10px] h-5 px-0 font-bold tracking-wider ${colorClass}`}>
        {event.type}
      </Badge>
      <div className="flex-1 min-w-0 break-all space-y-1">
        <span className="text-zinc-200 font-semibold block leading-tight">{event.key}</span>
        {event.value && (
          <span className="text-zinc-400 block break-all text-xs bg-zinc-950 p-1 rounded border border-zinc-800/50">
            {event.value}
          </span>
        )}
      </div>
    </div>
  );
}

export function EventStream() {
  const { events, connectionState } = useEventStream(50);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto scroll to latest event (which is actually at the top since we unshift, but let's keep it steady)
  // Actually, in useEventStream we do [event, ...prev], so events[0] is the newest.
  // This means the newest is at the top. We don't need to auto-scroll to bottom.

  return (
    <Card className="bg-zinc-900 border-zinc-800 flex flex-col h-full overflow-hidden">
      <CardHeader className="py-3 px-4 border-b border-zinc-800 bg-zinc-950/50 flex flex-row items-center justify-between space-y-0 sticky top-0 z-10">
        <div className="flex items-center gap-2">
          <Activity className="h-4 w-4 text-zinc-400" />
          <CardTitle className="text-sm font-semibold text-zinc-200">
            Live Event Stream
          </CardTitle>
        </div>
        <div className="flex items-center gap-2">
          <span className="relative flex h-2 w-2">
            {connectionState === "connected" && (
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            )}
            <span
              className={`relative inline-flex rounded-full h-2 w-2 ${
                connectionState === "connected"
                  ? "bg-green-500"
                  : connectionState === "connecting"
                  ? "bg-yellow-500 animate-pulse"
                  : "bg-red-500"
              }`}
            ></span>
          </span>
          <span
            className={`text-xs font-mono uppercase tracking-wider ${
              connectionState === "connected"
                ? "text-zinc-400"
                : connectionState === "connecting"
                ? "text-yellow-500"
                : "text-red-500"
            }`}
          >
            {connectionState}
          </span>
        </div>
      </CardHeader>
      <CardContent className="p-0 flex-1 overflow-y-auto" ref={scrollRef}>
        <div className="flex flex-col p-2 space-y-1">
          {events.length === 0 && (
            <div className="flex flex-col items-center justify-center p-8 text-zinc-500 space-y-4">
              <Activity className="h-8 w-8 opacity-20" />
              <p className="text-sm font-mono tracking-widest uppercase">Waiting for events...</p>
            </div>
          )}
          {events.map((event, i) => (
            <EventRow key={`${event.timestamp}-${i}`} event={event} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
