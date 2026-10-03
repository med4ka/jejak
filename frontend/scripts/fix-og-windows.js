#!/usr/bin/env node
// =====================================================================
// WINDOWS FIX: next/dist/compiled/@vercel/og/index.node.js
//
// That file reads its bundled assets (default font + yoga/resvg wasm) with:
//     fileURLToPath(join(import.meta.url, "../aset"))
// path.join uses win32 semantics → the file:/// URL becomes ".\file:\C:\..."
// → fileURLToPath throws ERR_INVALID_URL → EVERY route that uses
// ImageResponse (app/icon.jsx, app/u/[username]/opengraph-image.jsx)
// responds 500 on Windows only. See vercel/next.js#74385.
//
// Correct pattern (used by the edge variant and upstream @vercel/og):
//     fileURLToPath(new URL("../aset", import.meta.url))
//
// Idempotent: if the old pattern is absent (e.g. Next has already shipped a
// fix), this script changes nothing. Invoked automatically by
// "postinstall" (package.json) on every npm install: run manually when
// needed:
//     node scripts/fix-og-windows.js
// =====================================================================

const fs = require("fs");
const path = require("path");

const target = path.join(
  __dirname,
  "..",
  "node_modules",
  "next",
  "dist",
  "compiled",
  "@vercel",
  "og",
  "index.node.js"
);

if (!fs.existsSync(target)) {
  // Dependencies not yet installed: skip; the next postinstall run will
  // handle it.
  process.exit(0);
}

const source = fs.readFileSync(target, "utf8");

// join(import.meta.url, "../x") → path.join treats the URL as a directory
// sequence, so "../x" pops the FILE NAME → result ".../og/x"
// (sibling of index.node.js). new URL("../x", base) = DIRECTORY context →
// goes up one more level (".../@vercel/x", wrong). Hence the conversion to
// "./x" (file sibling): exactly equivalent to the old join result.
// Two patterns are caught: the original (join) code and a half-completed
// conversion.
const reJoin = /fileURLToPath\(join\(import\.meta\.url, "\.\.\/([^"]+)"\)\)/g;
const reUrl = /fileURLToPath\(new URL\("\.\.\/([^"]+)", import\.meta\.url\)\)/g;
const toSibling = (match, file) => `fileURLToPath(new URL("./${file}", import.meta.url))`;

let patched = source.replace(reJoin, toSibling);
patched = patched.replace(reUrl, toSibling);
const count =
  (source.match(reJoin) || []).length + (source.match(reUrl) || []).length;

if (count > 0) {
  fs.writeFileSync(target, patched, "utf8");
  console.log(`[fix-og-windows] mempatch ${count} pembacaan aset: ${target}`);
}
