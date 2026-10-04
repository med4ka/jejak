import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy "check health now" -> Go POST /api/links/{short_code}/check-health.
// Session cookie forwarded: Go authenticates, verifies ownership (404 for a
// foreign code) and rate-limits (10/min per IP) before the live probe.
export async function POST(req, props) {
  const params = await props.params;
  const shortCode = params.short_code;
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/links/${shortCode}/check-health`, {
      method: "POST",
      headers: { Cookie: cookie },
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
