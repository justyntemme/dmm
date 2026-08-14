#!/usr/bin/env node

import process from "node:process";
import { createServer } from "../web/node_modules/vite/dist/node/index.js";
import { render } from "../web/node_modules/svelte/src/server/index.js";

function requireMarkup(body, pattern, message) {
  if (!pattern.test(body)) {
    throw new Error(`${message}\nRendered markup:\n${body}`);
  }
}

const vite = await createServer({
  root: new URL("../web", import.meta.url).pathname,
  mode: "production",
  appType: "custom",
  server: { middlewareMode: true },
});

try {
  const module = await vite.ssrLoadModule("/src/App.svelte");
  const { body } = render(module.default);

  requireMarkup(body, /<main class="phone-shell">/, "phone application shell did not render");
  requireMarkup(body, /<header class="phone-topbar">/, "phone top bar did not render");
  requireMarkup(body, /aria-label="Open menu"/, "primary navigation control did not render");
  requireMarkup(body, /aria-label="Settings"/, "settings control did not render");
  requireMarkup(body, /<p>Running Now<\/p>/, "running-game context did not render");
  requireMarkup(body, /<h1>Decky Mod Manager<\/h1>/, "application title did not render");
  requireMarkup(body, /<section class="phone-empty"><h2>Loading\.\.\.<\/h2><\/section>/, "initial loading state did not render");
} finally {
  await vite.close();
}
