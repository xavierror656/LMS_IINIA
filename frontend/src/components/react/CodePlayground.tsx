import { useEffect, useRef, useState } from "react";
import { EditorView } from "@codemirror/view";
import { EditorState, Compartment } from "@codemirror/state";
import { basicSetup } from "codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { python } from "@codemirror/lang-python";
import { Play, Square, Trash2 } from "lucide-react";
import { connectCode, type Language } from "../../lib/websocket";
export default function CodePlayground({
  lessonId,
  language: initialLanguage,
  starter,
}: {
  lessonId: number;
  language: Language;
  starter: string;
}) {
  const container = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const languageSlot = useRef(new Compartment());
  const client = useRef<ReturnType<typeof connectCode> | null>(null);
  const active = useRef("");
  const request = useRef("");
  const lastSeq = useRef(0);
  const [language, setLanguage] = useState(initialLanguage);
  const [connected, setConnected] = useState(false);
  const [running, setRunning] = useState(false);
  const [accepted, setAccepted] = useState(false);
  const [retry, setRetry] = useState(0);
  const [lines, setLines] = useState<{ text: string; kind: string }[]>([]);
  const add = (text: string, kind = "stdout") =>
    setLines((old) =>
      [...old, { text: text.slice(0, 2048), kind }].slice(-100),
    );
  useEffect(() => {
    if (!container.current) return;
    const editor = new EditorView({
      parent: container.current,
      state: EditorState.create({
        doc: starter,
        extensions: [
          basicSetup,
          languageSlot.current.of(
            initialLanguage === "python" ? python() : javascript(),
          ),
          EditorView.lineWrapping,
          EditorView.contentAttributes.of({
            "aria-label": "Editor de código. Escape y después Tab para salir.",
          }),
        ],
      }),
    });
    view.current = editor;
    return () => {
      editor.destroy();
      view.current = null;
    };
  }, [starter, initialLanguage]);
  useEffect(() => {
    const ws = connectCode(
      (event) => {
        if (event.type === "run.failed") {
          add(event.payload.message ?? "No se pudo iniciar.", "stderr");
          if (event.requestId === request.current && !active.current) {
            setRunning(false);
            setAccepted(false);
          }
          return;
        }
        if (event.requestId !== request.current || event.seq <= lastSeq.current)
          return;
        lastSeq.current = event.seq;
        if (event.type === "run.accepted") {
          active.current = event.runId ?? "";
          setAccepted(true);
          setRunning(true);
        } else if (event.type === "run.finished") {
          add(
            event.payload.status === "cancelled"
              ? "Simulación detenida."
              : "Simulación finalizada.",
          );
          active.current = "";
          setRunning(false);
          setAccepted(false);
        } else
          add(
            event.payload.text ?? "",
            event.type === "run.stderr" ? "stderr" : "stdout",
          );
      },
      (status) => {
        setConnected(status === "connected");
        if (status === "disconnected") {
          setRunning(false);
          setAccepted(false);
          active.current = "";
          add(
            "Conexión cerrada. Puedes reconectar; tu ejecución no se reenviará.",
            "stderr",
          );
        }
      },
    );
    client.current = ws;
    return () => {
      if (active.current) ws.cancel(active.current);
      ws.close();
      client.current = null;
      active.current = "";
    };
  }, [lessonId, retry]);
  function run() {
    const code = view.current?.state.doc.toString() ?? "";
    if (new TextEncoder().encode(code).length > 16384) {
      add("Tu código supera el límite de 16 KiB.", "stderr");
      return;
    }
    try {
      lastSeq.current = 0;
      request.current = client.current!.start(lessonId, language, code);
      setRunning(true);
    } catch (e) {
      add(e instanceof Error ? e.message : "No se pudo conectar.", "stderr");
    }
  }
  return (
    <section
      className="panel playground card card-border bg-base-100"
      aria-label="Laboratorio de código"
    >
      <div>
        <span className="badge badge-soft badge-primary">Ejecución simulada</span>
        <p className="alert alert-info text-sm">
          El código que escribes no se ejecuta. Esta consola muestra mensajes de
          demostración enviados por el servidor.
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <label htmlFor="language">Lenguaje</label>
        <select
          className="select"
          id="language"
          value={language}
          disabled={running}
          onChange={(e) => {
            const value = e.target.value as Language;
            setLanguage(value);
            view.current?.dispatch({
              effects: languageSlot.current.reconfigure(
                value === "python" ? python() : javascript(),
              ),
            });
          }}
        >
          <option value="javascript">JavaScript</option>
          <option value="python">Python</option>
        </select>
        <span role="status" className="text-sm opacity-70">
          {connected ? "● Conectado" : "○ Desconectado"}
        </span>
        {!connected && (
          <button
            className="btn btn-outline"
            onClick={() => setRetry((n) => n + 1)}
          >
            Reconectar
          </button>
        )}
      </div>
      <div className="editor" ref={container} />
      <div className="flex flex-wrap items-center gap-3">
        <button
          className="btn btn-primary"
          onClick={run}
          disabled={!connected || running}
        >
          <Play size={18} aria-hidden="true" />
          Ejecutar
        </button>
        <button
          className="btn btn-outline"
          onClick={() => client.current?.cancel(active.current)}
          disabled={!running || !accepted}
        >
          <Square size={18} aria-hidden="true" />
          Detener
        </button>
        <button className="btn btn-outline" onClick={() => setLines([])}>
          <Trash2 size={18} aria-hidden="true" />
          Limpiar consola
        </button>
      </div>
      <div
        className="console"
        role="log"
        aria-label="Consola de simulación"
        aria-live="polite"
        tabIndex={0}
      >
        {lines.length === 0
          ? "La consola está lista para explorar."
          : lines.map((line, i) => (
              <p className={line.kind} key={i}>
                {line.text}
              </p>
            ))}
      </div>
    </section>
  );
}
