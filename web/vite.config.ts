import { defineConfig } from "vite";
import solidPlugin from "vite-plugin-solid";

export default defineConfig({
  plugins: [solidPlugin()],
  server: {
    port: 3000,
    allowedHosts: true,
    proxy: {
      "/aot.api.v1.AOTService": {
        target: "http://localhost:50055",
        changeOrigin: true,
      },
    },
  },
  build: {
    target: "esnext",
  },
});
