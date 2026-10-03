/** @type {import('next').NextConfig} */
const net = require("node:net");

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// SAFETY (polish 2026-09-30): `next dev` and `next start` share the .next
// folder. Running both at once lets dev overwrite the prod artifacts (and
// vice versa), which can turn certain routes into 500s in production. This
// config runs at build/start; the port probe only fires during `next build`,
// because during `next start` the port may be bound by the server process
// itself (a probe would be a false positive) - such collisions are already
// reported by Next as EADDRINUSE.
function warnIfServerAlreadyRunning(port) {
  if (process.env.NODE_ENV !== "production") return;
  if (!process.argv.includes("build")) return;
  const socket = net.createConnection({ port, host: "127.0.0.1" });
  let settled = false;
  const settle = (busy) => {
    if (settled) return;
    settled = true;
    socket.destroy();
    if (busy) {
      console.warn(
        `\n[next] PERINGATAN: port ${port} sudah terisi. ` +
          `Kalau itu "next dev", matikan dulu sebelum build/start: ` +
          `dev dan start share folder .next, jadi keduanya bisa saling ` +
          `menimpa (route jadi 500). Jalankan satu mode saja.\n`
      );
    }
  };
  socket.setTimeout(500, () => settle(false));
  socket.on("connect", () => settle(true));
  socket.on("error", () => settle(false));
  socket.unref();
}
warnIfServerAlreadyRunning(3000);

module.exports = {
  // Avatar uploads are stored locally by Go (POST /api/profile/avatar); the DB
  // keeps the relative path "/uploads/...". This rewrite lets avatar images
  // load from any page (dashboard + public page) through the Next origin,
  // which forwards to the API - no cookie or session is required to read this
  // static file.
  async rewrites() {
    return [
      {
        source: "/uploads/:path*",
        destination: `${GO_API_URL}/uploads/:path*`,
      },
    ];
  },
};