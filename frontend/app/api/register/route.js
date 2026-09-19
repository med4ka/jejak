import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy register -> Go. PENTING: teruskan header Set-Cookie dari Go ke
// browser — tanpa ini sesi login tidak pernah sampai ke client karena
// NextResponse.json() tidak meneruskan header itu otomatis.
export async function POST(req) {
  const body = await req.json();

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const text = await goRes.text();
  const res = new NextResponse(text, {
    status: goRes.status,
    headers: { "Content-Type": "application/json" },
  });
  const setCookie = goRes.headers.get("set-cookie");
  if (setCookie) {
    res.headers.set("set-cookie", setCookie);
  }
  return res;
}
