import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy same-origin -> Go API supaya browser tidak kena CORS.
// Browser hanya bicara ke Next; Next yang teruskan ke Go server-side.
// Cookie sesi ikut diteruskan supaya link yang dibuat user login otomatis
// ke-assign ke creator-nya (tanpa ini semua link dashboard jadi anonim).
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
    goRes = await fetch(`${GO_API_URL}/api/shorten`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Cookie: cookie },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const text = await goRes.text();
  if (!goRes.ok) {
    return NextResponse.json({ error: text || "Go API error" }, { status: goRes.status });
  }
  return NextResponse.json({ shortUrl: text }, { status: 201 });
}
