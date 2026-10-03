import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy for the caller's own profile -> Go (GET read, PUT update). The session
// cookie MUST be forwarded inbound (otherwise Go answers 401), and the reply
// Set-Cookie is forwarded outbound (symmetric with the other auth routes).
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
