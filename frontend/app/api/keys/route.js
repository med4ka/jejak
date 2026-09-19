import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy API keys -> Go (/api/keys): GET list, POST generate. Cookie sesi
// diteruskan (generate/list WAJIB login via session — bukan API key lain).
async function proxyKeys(req, method) {
  const cookie = req.headers.get("cookie") || "";
  let body;
  if (method === "POST") {
    try {
      body = await req.text();
    } catch {
      return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
    }
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/keys`, {
      method,
      headers: { "Content-Type": "application/json", Cookie: cookie },
      body,
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
  return proxyKeys(req, "GET");
}

export async function POST(req) {
  return proxyKeys(req, "POST");
}