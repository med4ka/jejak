import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy email change -> Go PUT /api/account/email. The session cookie is
// forwarded (Go answers 401 without a login; the reply Set-Cookie is forwarded
// outbound: same pattern as the profile proxy).
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
    goRes = await fetch(`${GO_API_URL}/api/account/email`, {
      method: "PUT",
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
