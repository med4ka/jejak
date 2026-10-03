import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy log out on all devices -> Go POST /api/account/logout-all. The session
// cookie is forwarded (Go revokes every session of the account) and the reply
// clear Set-Cookie is forwarded outbound: same pattern as the plain logout
// proxy.
export async function POST(req) {
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/account/logout-all`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Cookie: cookie },
      body: JSON.stringify({}),
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
