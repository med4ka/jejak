import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy for in-depth analytics -> Go GET /api/analytics/breakdown?kind=&range=.
// The query string is forwarded intact (kind device|referrer + range
// 7d|30d|90d); the session cookie goes along so Go answers 401 for anonymous
// callers and scopes the figures to the caller's own creator account.
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";
  const qs = new URL(req.url).search || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/analytics/breakdown${qs}`, {
      headers: { Cookie: cookie },
      cache: "no-store",
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const text = await goRes.text();
  return new NextResponse(text, {
    status: goRes.status,
    headers: { "Content-Type": "application/json" },
  });
}
