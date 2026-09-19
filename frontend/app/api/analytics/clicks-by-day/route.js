import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy analytics -> Go GET /api/analytics/clicks-by-day. Cookie sesi
// diteruskan (Go menolak tanpa login + memfilter ke link milik sendiri).
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/analytics/clicks-by-day`, {
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
