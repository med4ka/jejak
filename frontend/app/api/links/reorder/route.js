import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy reorder -> Go PUT /api/links/reorder. Cookie sesi diteruskan
// (Go menolak tanpa login + memverifikasi kepemilikan tiap baris).
export async function PUT(req) {
  const cookie = req.headers.get("cookie") || "";
  let body;
  try {
    body = await req.text();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/links/reorder`, {
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
