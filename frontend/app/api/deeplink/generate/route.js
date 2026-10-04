import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Same-origin proxy -> Go API (same pattern as app/api/shorten/route.js).
// Pure lookup, no session involved: the response is passed through verbatim
// (JSON success or plain-text error body with the original status).
export async function POST(req) {
  let body;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/deeplink/generate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
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
