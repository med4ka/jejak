import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy edit link -> Go PUT /api/links/{short_code}. Cookie sesi diteruskan
// (Go menolak tanpa login + memverifikasi kepemilikan — bukan milik = 404).
// Segmen statis "reorder" menang atas route dinamis ini di Next, jadi
// /api/links/reorder tetap pakai proxy-nya sendiri.
export async function PUT(req, { params }) {
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