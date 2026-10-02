import { beforeEach, expect, it, vi } from "vitest";
import {
  clearProgress,
  completeReading,
  progressStore,
  refreshProgress,
} from "./progress";
const progress = {
  userId: 1,
  alias: "Luna",
  xp: 25,
  level: 1,
  stars: 1,
  gems: 2,
  lives: 5,
  completed: 1,
};
beforeEach(() => {
  clearProgress();
  vi.restoreAllMocks();
});
it("LMS008 loads canonical progress and logout clears personal state", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response(JSON.stringify(progress))),
  );
  await refreshProgress();
  expect(progressStore.get().progress).toEqual(progress);
  clearProgress();
  expect(progressStore.get().progress).toBeNull();
  expect(progressStore.get().hydrated).toBe(false);
});
it("LMS008 ignores an in-flight response after logout", async () => {
  let resolve!: (r: Response) => void;
  vi.stubGlobal(
    "fetch",
    vi.fn(() => new Promise<Response>((r) => (resolve = r))),
  );
  const pending = refreshProgress();
  clearProgress();
  resolve(new Response(JSON.stringify(progress)));
  await pending;
  expect(progressStore.get().progress).toBeNull();
});
it("LMS008 outage is recoverable and never invents scores", async () => {
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("offline")));
  await refreshProgress();
  expect(progressStore.get().error).toContain("No podemos conectar");
  expect(progressStore.get().progress).toBeNull();
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response(JSON.stringify(progress))),
  );
  await refreshProgress();
  expect(progressStore.get().error).toBeNull();
});
it("LMS004 completion uses the server response", async () => {
  const fetch = vi
    .fn()
    .mockResolvedValue(new Response(JSON.stringify(progress)));
  vi.stubGlobal("fetch", fetch);
  await completeReading(1);
  expect(fetch.mock.calls[0]?.[0]).toBe("/api/v1/lessons/1/complete");
  expect(progressStore.get().progress?.xp).toBe(25);
});
