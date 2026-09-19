import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy profil milik sendiri -> Go (GET baca, PUT update). Cookie sesi
// WAJIB diteruskan masuk (tanpa ini Go menjawab 401), dan Set-Cookie
// balasan diteruskan keluar (simetris dengan route auth lain).
async function proxyProfile(req, method) {
  const cookie = req.headers.get("cookie") || "";
  let body;
  if (method !== "GET") {
    try {
      body = await req.text();
    } catch {
      return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
    }
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/profile`, {
      method,
      headers: { "Content-Type": "application/json", Cookie: cookie },
      body: method === "GET" ? undefined : body,
      cache: "no-store",
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

export async function GET(req) {
  return proxyProfile(req, "GET");
}

export async function PUT(req) {
  return proxyProfile(req, "PUT");
}
