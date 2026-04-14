import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createRequire } from "node:module";

import { build } from "esbuild";

const require = createRequire(import.meta.url);
const rootDir = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.resolve(rootDir, "../internal/tools/assets");

await fs.mkdir(outDir, { recursive: true });

await build({
  entryPoints: [path.resolve(rootDir, "src/render-pptx.ts")],
  bundle: true,
  platform: "node",
  format: "cjs",
  target: "node20",
  external: ["playwright-core"],
  outfile: path.resolve(outDir, "pptx-renderer.cjs"),
});

const domModulePath = require.resolve("dom-to-pptx");
const domBundlePath = path.join(path.dirname(domModulePath), "dom-to-pptx.bundle.js");
await fs.copyFile(domBundlePath, path.resolve(outDir, "dom-to-pptx.bundle.js"));
