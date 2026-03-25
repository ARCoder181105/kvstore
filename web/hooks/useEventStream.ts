"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import type { StoreEvent } from "@/lib/types";

type ConnectionState = "connecting" | "connected" | "disconnected";

export function useEventStream(maxEvents = 100) {
  const [events, setEvents] = useState<StoreEvent[]>([]);
  const [connectionState, setConnectionState] =
    useState<ConnectionState>("connecting");
  const wsRef = useRef<WebSocket | null>(null);
  const retryTimeoutRef = useRef<NodeJS.Timeout | undefined>(undefined);
  const retryDelayRef = useRef(1000);

  const connect = useCallback(() => {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL;
    if (!wsUrl) return;

    const ws = new WebSocket(`${wsUrl}/ws/events`);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnectionState("connected");
      retryDelayRef.current = 1000;
    };

    ws.onmessage = (e) => {
      try {
        const event: StoreEvent = JSON.parse(e.data);
        setEvents((prev) => [event, ...prev].slice(0, maxEvents));
      } catch {
        // malformed message — ignore
      }
    };

    ws.onclose = () => {
      setConnectionState("disconnected");
      retryTimeoutRef.current = setTimeout(() => {
        retryDelayRef.current = Math.min(retryDelayRef.current * 2, 30000);
        connect();
      }, retryDelayRef.current);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [maxEvents]);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(retryTimeoutRef.current);
      wsRef.current?.close();
    };
  }, [connect]);

  return { events, connectionState };
}
