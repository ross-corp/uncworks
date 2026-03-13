import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 3000,
    allowedHosts: true,
    proxy: {
      "/aot.api.v1.AOTService": {
        target: "http://localhost:50055",
        changeOrigin: true,
      },
    },
  },
});
