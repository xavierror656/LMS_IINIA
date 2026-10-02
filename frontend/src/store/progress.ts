import { atom } from "nanostores";
import { api, ApiError, type Progress } from "../lib/api";
export interface ProgressState {
  hydrated: boolean;
  loading: boolean;
  progress: Progress | null;
  error: string | null;
}
const initial = (): ProgressState => ({
  hydrated: false,
  loading: false,
  progress: null,
  error: null,
});
export const progressStore = atom<ProgressState>(initial());
let generation = 0;
export function clearProgress() {
  generation++;
  progressStore.set(initial());
}
export async function refreshProgress() {
  if (typeof window === "undefined") return;
  const ticket = ++generation;
  progressStore.set({ ...progressStore.get(), loading: true, error: null });
  try {
    const p = await api<Progress>("/me/progress");
    if (ticket === generation)
      progressStore.set({
        hydrated: true,
        loading: false,
        progress: p,
        error: null,
      });
  } catch (e) {
    if (ticket !== generation) return;
    progressStore.set({
      hydrated: true,
      loading: false,
      progress: null,
      error:
        e instanceof ApiError && e.status === 401
          ? null
          : e instanceof Error
            ? e.message
            : "No se pudo cargar tu progreso.",
    });
  }
}
export async function completeReading(id: number) {
  const ticket = ++generation;
  const p = await api<Progress>(`/lessons/${id}/complete`, {
    method: "POST",
    body: "{}",
  });
  if (ticket === generation)
    progressStore.set({
      hydrated: true,
      loading: false,
      progress: p,
      error: null,
    });
  return p;
}
