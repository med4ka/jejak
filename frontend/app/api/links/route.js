import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy same-origin -> Go API GET /api/links (link milik user yang login;
// sesi diteruskan supaya server bisa scope per-creator).
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";
  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/links`, {
      cache: "no-store",
      headers: cookie ? { cookie } : {},
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const data = await goRes.json();
  if (!goRes.ok) {
    return NextResponse.json({ error: "Go API error" }, { status: goRes.status });
  }
  return NextResponse.json(data);
}
