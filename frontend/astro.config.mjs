import { defineConfig } from "astro/config";
import react from "@astrojs/react";
import node from "@astrojs/node";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";
export default defineConfig({
  output: "server",
  devToolbar: { enabled: false },
  adapter: node({ mode: "standalone" }),
  integrations: [react()],
  vite: {
    plugins: [tailwindcss()],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
    server: {
      strictPort: process.env.PUBLIC_DEMO_MODE === "true",
      proxy: {
        "/api": {
          target: process.env.API_INTERNAL_URL || "http://127.0.0.1:8080",
          changeOrigin: false,
        },
        "/ws": {
          target: process.env.API_INTERNAL_URL || "http://127.0.0.1:8080",
          ws: true,
          changeOrigin: false,
        },
      },
    },
  },
  server: { port: 4321, host: true },
});
