import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    // The webview is a modern WebKit, so we can target recent syntax.
    target: "es2022",
    chunkSizeWarningLimit: 1500,
  },
  plugins: [vue(), wails("./bindings")],
});
