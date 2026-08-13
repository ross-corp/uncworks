/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from 'node:path';
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin';
import { playwright } from '@vitest/browser-playwright';
// import.meta.dirname keeps the config loadable by Vite's native config loader,
// which __dirname is not.
const dirname = import.meta.dirname;

// More info at: https://storybook.js.org/docs/next/writing-tests/integrations/vitest-addon
export default defineConfig({
  plugins: [tailwindcss(), react()],
  server: {
    host: "0.0.0.0",
    port: 3000,
    allowedHosts: true,
    proxy: {
      "/api/": {
        target: process.env.VITE_API_URL ?? "http://localhost:50055",
        changeOrigin: true,
      },
      "/aot.api.v1.AOTService": {
        target: process.env.VITE_API_URL ?? "http://localhost:50055",
        changeOrigin: true,
      },
    }
  },
  test: {
    projects: [{
      extends: true,
      plugins: [
      // The plugin will run tests for the stories defined in your Storybook config
      // See options at: https://storybook.js.org/docs/next/writing-tests/integrations/vitest-addon#storybooktest
      storybookTest({
        configDir: path.join(dirname, '.storybook')
      })],
      test: {
        name: 'storybook',
        browser: {
          enabled: true,
          headless: true,
          provider: playwright({}),
          instances: [{
            browser: 'chromium'
          }]
        },
        setupFiles: ['.storybook/vitest.setup.ts']
      }
    }]
  }
});