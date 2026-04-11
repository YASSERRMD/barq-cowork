import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// Vite config for Tauri dev/build.
// https://vitejs.dev/config/
export default defineConfig(async () => ({
  plugins: [react()],

  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },

  // Tauri uses localhost; prevent Vite from obscuring errors behind the network.
  clearScreen: false,

  server: {
    port: 1420,
    strictPort: true,
    watch: {
      // Ignore the Rust side so hot-reload is fast.
      ignored: ["**/src-tauri/**"],
    },
  },

  // Build target compatible with Tauri's webview.
  build: {
    target: process.env.TAURI_ENV_PLATFORM === "windows" ? "chrome105" : "safari13",
    minify: process.env.TAURI_ENV_DEBUG ? false : "esbuild",
    sourcemap: !!process.env.TAURI_ENV_DEBUG,
  },
}));
