import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { readFileSync } from "node:fs";

// The version lives only in build/config.yml (see scripts/version.sh).
function appVersion(): string {
  const yml = readFileSync("../build/config.yml", "utf8");
  return yml.match(/^info:[^]*?^ {2}version: "([^"]+)"/m)?.[1] ?? "dev";
}

export default defineConfig({
  // GitHub Pages serves docs/ under /<repo>/; relative URLs work there and
  // from a local file alike.
  base: "./",
  plugins: [vue()],
  // The screenshots and icon already sit in docs/, so they are referenced, not
  // copied; `make site` clears the old build before this one.
  publicDir: false,
  build: { outDir: "../docs", emptyOutDir: false },
  define: { __APP_VERSION__: JSON.stringify(appVersion()) },
});
