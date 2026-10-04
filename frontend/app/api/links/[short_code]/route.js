import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy link edit -> Go PUT /api/links/{short_code}. The session cookie is
// forwarded (Go rejects unauthenticated requests and verifies ownership: a
// link not owned returns 404). The static "reorder" segment wins over this
// dynamic route in Next, so /api/links/reorder keeps its own proxy.
export async function PUT(req, props) {
  const params = await props.params;
  const shortCode = params.short_code;
  const cookie = req.headers.get("cookie") || "";
  let body;
  try {
    body = await req.text();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/links/${shortCode}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", Cookie: cookie },
      body,
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