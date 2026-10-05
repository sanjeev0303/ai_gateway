"use client";

import * as React from "react";

import { CONTROL_API_BASE } from "@/lib/control-api";

/**
 * Live activity over server-sent events.
 *
 * The payload mirrors the control API's `ActivityEvent`. It deliberately has no
 * prompt or completion field, so no request content can reach the browser even
 * by accident — the dashboard shows routing metadata only.
 */
export interface LiveActivityEvent {
  request_id: string;
  organization_id: string;
  user_id?: string;
  user_name?: string;
  service_account?: string;
  project_id?: string;
  project_name?: string;
  team_id?: string;
  provider: string;
  model: string;
  status: string;
  started_at: string;
  current_state: string;
}

export type LiveStatus = "connecting" | "live" | "reconnecting" | "offline";

interface UseLiveActivityOptions {
  /** Server-rendered snapshot, so the first paint is already populated. */
  initialEvents: readonly LiveActivityEvent[];
  /** Ring-buffer size. Older rows fall off the top. */
  maxRows?: number;
}

const BASE_DELAY_MS = 1_000;
const MAX_DELAY_MS = 30_000;

function startedDesc(a: LiveActivityEvent, b: LiveActivityEvent): number {
  return b.started_at.localeCompare(a.started_at);
}

/**
 * Merge one streamed event into the row set. A request can stream more than
 * once as its state advances (active, then completed), so rows are keyed by
 * request id and the newest version wins rather than being duplicated.
 */
function mergeEvent(
  previous: readonly LiveActivityEvent[],
  incoming: LiveActivityEvent,
  maxRows: number,
): LiveActivityEvent[] {
  const withoutIncoming = previous.filter(
    (event) => event.request_id !== incoming.request_id,
  );
  return [incoming, ...withoutIncoming].sort(startedDesc).slice(0, maxRows);
}

/**
 * Parse one SSE frame. The backend opens with `event: ping` as a keepalive and
 * then sends unnamed `data:` frames, so a frame without an event name is a real
 * activity event.
 */
function parseFrame(frame: string): { event: string; data: string } | null {
  let event = "message";
  const data: string[] = [];

  for (const line of frame.split("\n")) {
    if (line.startsWith(":")) continue;
    if (line.startsWith("event:")) {
      event = line.slice("event:".length).trim();
    } else if (line.startsWith("data:")) {
      data.push(line.slice("data:".length).trim());
    }
  }

  if (data.length === 0) return null;
  return { event, data: data.join("\n") };
}

/**
 * Subscribe to `/v1/activity/stream` and keep a live row set.
 *
 * Uses `fetch` with a streaming body reader rather than `EventSource` because
 * teardown has to be abortable — `EventSource` exposes no abort handle, so an
 * unmounted component would keep its socket and retry loop alive. The browser
 * never polls: the only database read is the server-rendered snapshot passed in
 * as `initialEvents`.
 */
export function useLiveActivity({
  initialEvents,
  maxRows = 50,
}: UseLiveActivityOptions): {
  events: readonly LiveActivityEvent[];
  status: LiveStatus;
} {
  const [events, setEvents] = React.useState<LiveActivityEvent[]>(() =>
    [...initialEvents].sort(startedDesc).slice(0, maxRows),
  );
  const [status, setStatus] = React.useState<LiveStatus>("connecting");

  // Kept in refs so the reconnect loop never closes over a stale render.
  const controllerRef = React.useRef<AbortController | null>(null);
  const timerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = React.useRef(0);
  const maxRowsRef = React.useRef(maxRows);
  maxRowsRef.current = maxRows;

  React.useEffect(() => {
    let disposed = false;

    const clearTimer = () => {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };

    const scheduleReconnect = () => {
      if (disposed) return;
      setStatus("reconnecting");
      // Exponential backoff, capped. Attempt 0 waits the base delay.
      const delay = Math.min(BASE_DELAY_MS * 2 ** attemptRef.current, MAX_DELAY_MS);
      attemptRef.current += 1;
      clearTimer();
      timerRef.current = setTimeout(() => {
        void connect();
      }, delay);
    };

    const applyEvent = (raw: string) => {
      let parsed: LiveActivityEvent;
      try {
        parsed = JSON.parse(raw) as LiveActivityEvent;
      } catch {
        // A truncated frame is not worth tearing the connection down for.
        return;
      }
      if (!parsed || typeof parsed.request_id !== "string") return;
      setEvents((previous) => mergeEvent(previous, parsed, maxRowsRef.current));
    };

    const connect = async () => {
      if (disposed) return;

      const controller = new AbortController();
      controllerRef.current = controller;

      try {
        const response = await fetch(`${CONTROL_API_BASE}/v1/activity/stream`, {
          method: "GET",
          credentials: "include",
          headers: { Accept: "text/event-stream" },
          signal: controller.signal,
        });

        if (!response.ok || !response.body) {
          scheduleReconnect();
          return;
        }

        // Connected: reset backoff so a later drop retries promptly.
        attemptRef.current = 0;
        setStatus("live");

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          // Frames are separated by a blank line. Tolerate CRLF from any proxy.
          buffer = buffer.replace(/\r\n/g, "\n");

          let boundary = buffer.indexOf("\n\n");
          while (boundary !== -1) {
            const frame = buffer.slice(0, boundary);
            buffer = buffer.slice(boundary + 2);
            const parsed = parseFrame(frame);
            if (parsed) {
              if (parsed.event === "ping") {
                attemptRef.current = 0;
                setStatus("live");
              } else {
                applyEvent(parsed.data);
              }
            }
            boundary = buffer.indexOf("\n\n");
          }
        }

        // Clean EOF: the server closed, so reconnect.
        scheduleReconnect();
      } catch (error) {
        if (disposed || (error as Error)?.name === "AbortError") return;
        scheduleReconnect();
      }
    };

    void connect();

    return () => {
      disposed = true;
      clearTimer();
      controllerRef.current?.abort();
      controllerRef.current = null;
    };
  }, []);

  return { events, status };
}
