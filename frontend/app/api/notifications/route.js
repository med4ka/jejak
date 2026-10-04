import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy notification feed -> Go GET /api/notifications (navbar bell). The
// session cookie is forwarded: Go scopes the feed to the logged-in creator
// and 401s anonymous callers (passed through unchanged).
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/notifications`, {
      method: "GET",
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
