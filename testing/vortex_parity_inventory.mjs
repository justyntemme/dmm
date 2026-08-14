#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFileSync } from "node:child_process";
import ts from "../web/node_modules/typescript/lib/typescript.js";

const expectedCommit = "c57894eb71af8234b58a6bd15ae5ab543eccac3a";
const sourceRoot = path.resolve(process.env.DMM_VORTEX_SOURCE || "/tmp/dmm-vortex");
const extensionsRoot = path.join(sourceRoot, "extensions");
const outputPath = path.resolve(process.env.DMM_VORTEX_INVENTORY || "testing/vortex-parity-inventory.json");
const checkOnly = process.argv.includes("--check");

function fail(message) {
  process.stderr.write(`vortex parity inventory: ${message}\n`);
  process.exit(1);
}

function sourceCommit() {
  try {
    return execFileSync("git", ["-C", sourceRoot, "rev-parse", "HEAD"], { encoding: "utf8" }).trim();
  } catch (error) {
    fail(`cannot inspect ${sourceRoot}: ${error.message}`);
  }
}

function walk(directory) {
  const files = [];
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const fullPath = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...walk(fullPath));
    else if (/\.(?:js|jsx|ts|tsx)$/.test(entry.name)) files.push(fullPath);
  }
  return files.sort();
}

function componentFor(relativePath) {
  const parts = relativePath.split("/");
  return parts[0] === "games" && parts.length > 1 ? `${parts[0]}/${parts[1]}` : parts[0];
}

function propertyChain(expression) {
  const parts = [];
  let current = expression;
  while (ts.isPropertyAccessExpression(current)) {
    parts.unshift(current.name.text);
    current = current.expression;
  }
  if (ts.isIdentifier(current)) parts.unshift(current.text);
  return parts;
}

function registrationSurface(expression) {
  const parts = propertyChain(expression);
  if (parts[0] !== "context") return "";
  const method = parts.at(-1) || "";
  return method.startsWith("register") ? method : "";
}

function lineNumber(sourceFile, node) {
  return sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1;
}

function scanFile(filePath, components) {
  const relativePath = path.relative(extensionsRoot, filePath).split(path.sep).join("/");
  const component = componentFor(relativePath);
  const source = fs.readFileSync(filePath, "utf8");
  const kind = filePath.endsWith("x") ? ts.ScriptKind.TSX : filePath.endsWith(".js") || filePath.endsWith(".jsx") ? ts.ScriptKind.JSX : ts.ScriptKind.TS;
  const sourceFile = ts.createSourceFile(relativePath, source, ts.ScriptTarget.Latest, true, kind);

  function visit(node) {
    if (ts.isCallExpression(node)) {
      const surface = registrationSurface(node.expression);
      if (surface) {
        if (!components.has(component)) components.set(component, new Map());
        const surfaces = components.get(component);
        if (!surfaces.has(surface)) surfaces.set(surface, []);
        surfaces.get(surface).push({ file: relativePath, line: lineNumber(sourceFile, node) });
      }
    }
    ts.forEachChild(node, visit);
  }
  visit(sourceFile);
}

if (!fs.existsSync(extensionsRoot)) fail(`missing Vortex extensions directory: ${extensionsRoot}`);
const commit = sourceCommit();
if (commit !== expectedCommit) fail(`source commit ${commit} does not match pinned ${expectedCommit}`);

const components = new Map();
for (const filePath of walk(extensionsRoot)) scanFile(filePath, components);

const inventory = {
  schema_version: 1,
  source_repository: "https://github.com/Nexus-Mods/Vortex",
  source_commit: expectedCommit,
  components: [...components.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([id, surfaces]) => ({
    id,
    surfaces: [...surfaces.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([name, calls]) => ({
      name,
      calls: calls.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line),
    })),
  })),
};
const rendered = `${JSON.stringify(inventory, null, 2)}\n`;

if (checkOnly) {
  if (!fs.existsSync(outputPath)) fail(`missing committed inventory ${outputPath}`);
  const current = fs.readFileSync(outputPath, "utf8");
  if (current !== rendered) fail(`committed inventory is stale; run node testing/vortex_parity_inventory.mjs`);
  process.stdout.write(`Vortex parity inventory verified: ${inventory.components.length} components\n`);
} else {
  fs.writeFileSync(outputPath, rendered);
  process.stdout.write(`Wrote ${outputPath}: ${inventory.components.length} components\n`);
}
