// The live demo on the website: the real frontend, with the Go services
// answered in the browser by demo/backend.js. `make site` builds it into
// docs/demo, next to the site GitHub Pages serves from docs/.
import { defineConfig, mergeConfig, type Plugin } from "vite";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import base from "./vite.config";

const appBindings = "bindings/github.com/cn8001/reforgepulse/internal/app";

// The runtime only sends a method's numeric ID; the generated bindings are
// the one place that maps names to IDs, so read them instead of copying IDs.
function bindingIDs(): Record<string, number> {
  const ids: Record<string, number> = {};
  for (const file of readdirSync(appBindings).filter((f) => f.endsWith("service.ts"))) {
    const service = file.replace(/\.ts$/, "");
    const src = readFileSync(join(appBindings, file), "utf8");
    for (const m of src.matchAll(/export function (\w+)\([^]*?\$Call\.ByID\((\d+)/g)) {
      ids[`${service}.${m[1]}`] = Number(m[2]);
    }
  }
  return ids;
}

// The version lives only in build/config.yml (see scripts/version.sh).
export function appVersion(): string {
  const yml = readFileSync("../build/config.yml", "utf8");
  return yml.match(/^info:[^]*?^ {2}version: "([^"]+)"/m)?.[1] ?? "dev";
}

function demo(): Plugin {
  const virtualID = "virtual:binding-ids";
  return {
    name: "reforgepulse-demo",
    resolveId: (id) => (id === virtualID ? "\0" + virtualID : null),
    load: (id) => (id === "\0" + virtualID ? `export default ${JSON.stringify(bindingIDs())}` : null),
    // "pre", or Vite has already collected the entry script from the page.
    transformIndexHtml: { order: "pre", handler: (html) => html.replace('src="/src/main.ts"', 'src="/demo/main.ts"') },
  };
}

export default mergeConfig(
  base,
  defineConfig({
    base: "./",
    plugins: [demo()],
    define: { __APP_VERSION__: JSON.stringify(appVersion()) },
    build: { outDir: "../docs/demo", emptyOutDir: true },
  }),
);
