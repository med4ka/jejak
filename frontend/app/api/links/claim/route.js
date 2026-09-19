import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy klaim -> Go POST /api/links/claim. Cookie sesi diteruskan
// (Go menolak tanpa login + hanya mengklaim baris yang masih NULL).
export async function POST(req) {
  const cookie = req.headers.get("cookie") || "";
  let body;
  try {
    body = await req.text();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/links/claim`, {
      method: "POST",
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
