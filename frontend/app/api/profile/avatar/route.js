import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy for the multipart POST avatar upload -> Go. The FormData body is
// forwarded raw (including the boundary in Content-Type) so Go can parse its
// magic bytes, and the session cookie MUST be forwarded so Go does not answer
// 401. The reply Set-Cookie is forwarded outbound, symmetric with the other
// auth routes.
export async function POST(req) {
  const cookie = req.headers.get("cookie") || "";
  const contentType = req.headers.get("content-type") || "";

  let rawBody;
  try {
    rawBody = await req.arrayBuffer();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/profile/avatar`, {
      method: "POST",
      headers: { "Content-Type": contentType, Cookie: cookie },
      body: rawBody,
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