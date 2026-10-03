import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy analytics -> Go GET /api/analytics/summary. The session cookie is
// forwarded (Go accepts the jejak_session cookie; an API key is also valid via
// Bearer).
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/analytics/summary`, {
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
