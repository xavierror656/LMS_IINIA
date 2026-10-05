import { useEffect, useRef, useState } from "react";
import { activities } from "../../config/h5p";
import { api } from "../../lib/api";
type XAPIEvent = {
  data?: {
    statement?: {
      verb?: { id?: string };
      result?: { score?: { scaled?: number } };
    };
  };
};
type Dispatcher = {
  on: (event: string, handler: (event: XAPIEvent) => void) => void;
  off: (event: string, handler: (event: XAPIEvent) => void) => void;
};
export default function H5PPlayer({
  lessonId,
  activity,
}: {
  lessonId: number;
  activity: string;
}) {
  const container = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState("Cargando actividad…");
  useEffect(() => {
    let disposed = false;
    let dispatcher: Dispatcher | undefined;
    const host = container.current;
    const config = activities[activity];
    const handler = (event: XAPIEvent) => {
      const statement = event.data?.statement;
      const verb = statement?.verb?.id?.split("/").pop();
      if (
        !verb ||
        !["completed", "answered", "passed", "failed"].includes(verb)
      )
        return;
      const score = statement?.result?.score?.scaled;
      void api(`/lessons/${lessonId}/attempts`, {
        method: "POST",
        body: JSON.stringify({
          verb,
          ...(typeof score === "number" && score >= 0 && score <= 1
            ? { score }
            : {}),
        }),
      })
        .then(() => {
          if (!disposed)
            setStatus(
              "Actividad reportada. No constituye una evaluación validada.",
            );
        })
        .catch(() => {
          if (!disposed)
            setStatus(
              "No se pudo guardar este intento. Puedes seguir explorando.",
            );
        });
    };
    if (!config?.enabled || !host) {
      setStatus("Actividad aún no configurada");
      return;
    }
    void (async () => {
      try {
        const { H5P } = await import("h5p-standalone");
        if (disposed) return;
        await new H5P(host, {
          h5pJsonPath: config.path,
          frameJs: "/h5p/player/frame.bundle.js",
          frameCss: "/h5p/player/styles/h5p.css",
          frame: true,
        });
        if (disposed) {
          host.replaceChildren();
          return;
        }
        dispatcher = (
          window as unknown as { H5P?: { externalDispatcher: Dispatcher } }
        ).H5P?.externalDispatcher;
        dispatcher?.on("xAPI", handler);
        setStatus("Explora la actividad a tu ritmo.");
      } catch {
        if (!disposed)
          setStatus(
            "No se pudo cargar la actividad. Tu docente puede revisar sus recursos.",
          );
      }
    })();
    return () => {
      disposed = true;
      dispatcher?.off("xAPI", handler);
      host.replaceChildren();
    };
  }, [lessonId, activity]);
  return (
    <section className="panel card card-border bg-base-100">
      <span className="badge badge-soft badge-primary">Actividad interactiva</span>
      <p role="status">{status}</p>
      <div className="h5p-container" ref={container} />
      <p className="text-sm opacity-70">
        Los intentos reportados por esta actividad no otorgan recompensas
        automáticamente.
      </p>
    </section>
  );
}
