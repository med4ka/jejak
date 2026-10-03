import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy for the QR ZIP download -> Go GET /api/links/qr-bulk.zip (link
// management feature, 2026-09-30). The path on the Next side deliberately
// omits ".zip" (/api/links/qr-bulk) for tidiness; Go uses the literal
// "/api/links/qr-bulk.zip": it must NEVER become the {short_code} wildcard
// (it would be swallowed by the link detail route).
// The session cookie is forwarded; ?tag= is forwarded as a filter. The binary
// body (application/zip) + Content-Disposition are forwarded intact.
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";
  const qs = new URL(req.url).search || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/links/qr-bulk.zip${qs}`, {
      headers: { Cookie: cookie },
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const buf = await goRes.arrayBuffer();
  const headers = new Headers();
  headers.set("Content-Type", goRes.headers.get("Content-Type") || "application/zip");
  const cd = goRes.headers.get("Content-Disposition");
  if (cd) {
    headers.set("Content-Disposition", cd);
  }
  return new NextResponse(buf, { status: goRes.status, headers });
}
