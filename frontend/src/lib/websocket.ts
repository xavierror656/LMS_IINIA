export type Language = "javascript" | "python";
export interface RunEvent {
  v: 1;
  type:
    | "run.accepted"
    | "run.stdout"
    | "run.stderr"
    | "run.finished"
    | "run.failed";
  requestId: string;
  runId?: string;
  seq: number;
  payload: {
    text?: string;
    status?: string;
    code?: string;
    message?: string;
    simulated?: boolean;
  };
}
export function connectCode(
  onEvent: (event: RunEvent) => void,
  onStatus: (status: "connected" | "disconnected") => void,
) {
  const url = new URL("/ws/code", window.location.href);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  const socket = new WebSocket(url);
  socket.onopen = () => onStatus("connected");
  socket.onclose = () => onStatus("disconnected");
  socket.onerror = () => onStatus("disconnected");
  socket.onmessage = (message) => {
    try {
      const event = JSON.parse(String(message.data)) as RunEvent;
      if (
        event.v === 1 &&
        typeof event.type === "string" &&
        event.type.startsWith("run.") &&
        typeof event.seq === "number" &&
        event.payload
      )
        onEvent(event);
    } catch {
      onStatus("disconnected");
      socket.close();
    }
  };
  return {
    start(lessonId: number, language: Language, code: string) {
      if (socket.readyState !== WebSocket.OPEN)
        throw new Error("Conecta de nuevo antes de ejecutar.");
      const requestId = crypto.randomUUID();
      socket.send(
        JSON.stringify({
          v: 1,
          type: "run.start",
          requestId,
          payload: { lessonId, language, code },
        }),
      );
      return requestId;
    },
    cancel(runId: string) {
      if (socket.readyState === WebSocket.OPEN)
        socket.send(
          JSON.stringify({
            v: 1,
            type: "run.cancel",
            requestId: crypto.randomUUID(),
            runId,
            payload: {},
          }),
        );
    },
    close() {
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      socket.close();
    },
  };
}
