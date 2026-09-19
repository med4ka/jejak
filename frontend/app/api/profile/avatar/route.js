import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy POST multipart upload avatar -> Go. Body FormData diteruskan mentah
// (termasuk boundary di Content-Type) supaya Go bisa parse magic bytes-nya,
// dan cookie sesi WAJIB diteruskan agar Go tidak menjawab 401. Set-Cookie
// balasan diteruskan keluar, simetris dengan route auth lain.
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