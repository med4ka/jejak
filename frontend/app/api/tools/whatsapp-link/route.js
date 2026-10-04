import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Same-origin proxy -> Go API (same pattern as app/api/shorten/route.js).
// The session cookie is forwarded so a logged-in creator owns the short link
// the WhatsApp tool creates; anonymous requests stay anonymous.
export async function POST(req) {
  let body;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/tools/whatsapp-link`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Cookie: cookie },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const text = await goRes.text();
  return new Response(text, {
    status: goRes.status,
    headers: { "Content-Type": "application/json" },
  });
}
