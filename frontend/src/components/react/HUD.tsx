import { useEffect, useRef, useState, type CSSProperties } from "react";
import type { Progress } from "../../lib/api";
import { useStore } from "@nanostores/react";
import { Star, Gem, Heart, Sparkles, Smile } from "lucide-react";
import { motion, useReducedMotion } from "framer-motion";
import {
  progressStore,
  refreshProgress,
  clearProgress,
} from "../../store/progress";
export default function HUD({ userId }: { userId: number }) {
  const state = useStore(progressStore);
  const reduced = useReducedMotion();
  const previous = useRef<Progress | null>(null);
  const [reward, setReward] = useState<{
    xp: number;
    stars: number;
    gems: number;
    level: number | null;
  } | null>(null);
  useEffect(() => {
    const current = state.progress;
    const before = previous.current;
    previous.current = current;
    if (!current || current.userId !== userId) {
      setReward(null);
      return;
    }
    // First hydration, an unchanged refresh and another user's data never celebrate.
    if (!before || before.userId !== current.userId || current.xp <= before.xp)
      return;
    setReward({
      xp: current.xp - before.xp,
      stars: Math.max(0, current.stars - before.stars),
      gems: Math.max(0, current.gems - before.gems),
      level: current.level > before.level ? current.level : null,
    });
  }, [state.progress, userId]);
  useEffect(() => {
    if (!reward) return;
    const timer = window.setTimeout(() => setReward(null), 3200);
    return () => window.clearTimeout(timer);
  }, [reward]);
  useEffect(() => {
    clearProgress();
    void refreshProgress();
    const refresh = () => void refreshProgress();
    document.addEventListener("astro:page-load", refresh);
    return () => {
      document.removeEventListener("astro:page-load", refresh);
      clearProgress();
    };
  }, [userId]);
  if (state.error)
    return (
      <aside className="hud" aria-label="Tu progreso">
        <span role="status">{state.error}</span>
        <button onClick={() => void refreshProgress()}>Reintentar</button>
      </aside>
    );
  if (!state.progress)
    return (
      <aside className="hud" aria-busy={state.loading}>
        Cargando tu mochila de logros…
      </aside>
    );
  const p = state.progress;
  return (
    <aside className="hud" aria-label="Tu progreso">
      <motion.div
        className="avatar"
        aria-hidden="true"
        animate={
          !reduced && reward
            ? { rotate: [0, -10, 10, 0], scale: [1, 1.12, 1] }
            : { rotate: 0, scale: 1 }
        }
        transition={{ duration: reduced ? 0 : 0.55 }}
      >
        <Smile size={34} strokeWidth={1.6} />
      </motion.div>
      <div className="hud-name">
        <strong>{p.alias}</strong>
        <span>
          Nivel {p.level} · {p.xp} XP
        </span>
        <div
          className="xp-track"
          role="progressbar"
          aria-label="Experiencia para el siguiente nivel"
          aria-valuenow={p.xp % 100}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuetext={`${p.xp % 100} de 100 XP para el siguiente nivel`}
        >
          <motion.span
            initial={false}
            animate={{ width: `${p.xp % 100}%` }}
            transition={{ duration: reduced ? 0 : 0.65, ease: "easeOut" }}
          />
        </div>
      </div>
      <motion.div
        className="hud-stats"
        initial={false}
        animate={!reduced && reward ? { scale: [1, 1.045, 1] } : { scale: 1 }}
        transition={{ duration: reduced ? 0 : 0.5 }}
      >
        <span>
          <Star aria-hidden="true" /> {p.stars} <small>estrellas</small>
        </span>
        <span>
          <Gem aria-hidden="true" /> {p.gems} <small>gemas</small>
        </span>
        <span>
          <Heart aria-hidden="true" /> {p.lives} <small>vidas</small>
        </span>
        <span>
          <Sparkles aria-hidden="true" /> {p.completed} <small>logros</small>
        </span>
      </motion.div>
      {reward && (
        <motion.div
          className="reward-toast"
          role="status"
          aria-live="polite"
          aria-atomic="true"
          initial={reduced ? false : { opacity: 0, y: 18, scale: 0.94 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: reduced ? 0 : 0.3 }}
        >
          {!reduced && (
            <div className="reward-sparks" aria-hidden="true">
              {Array.from({ length: 10 }, (_, i) => (
                <span
                  key={i}
                  style={
                    {
                      "--spark-x": `${((i % 5) - 2) * 36}px`,
                      "--spark-y": `${-45 - (i % 3) * 24}px`,
                      "--spark-delay": `${i * 25}ms`,
                      "--spark-turn": `${i % 2 ? 100 : -100}deg`,
                    } as CSSProperties
                  }
                >
                  ✦
                </span>
              ))}
            </div>
          )}
          <span className="reward-emblem" aria-hidden="true">
            <Star size={28} />
          </span>
          <div>
            <strong>
              {reward.level
                ? `¡Llegaste al nivel ${reward.level}!`
                : "¡Un descubrimiento más!"}
            </strong>
            <span>
              +{reward.xp} XP · +{reward.stars} estrellas · +{reward.gems} gemas
            </span>
            <small>Tu progreso está guardado.</small>
          </div>
        </motion.div>
      )}
    </aside>
  );
}
