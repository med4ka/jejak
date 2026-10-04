import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy password unlock POST /r/{code}/verify -> Go (link protection,
// migration 17). Without this route the password form (served through the
// GET /r/{code} proxy) would submit to a Next 404 and the visitor could
// never unlock a protected link from the site origin.
//
// The form body (application/x-www-form-urlencoded) and cookie are forwarded
// verbatim; redirect: "manual" keeps Go's 303 (PRG) + Set-Cookie intact so
// the BROWSER performs the redirect and stores the 1-hour link-access cookie.
// Forwarded headers mirror app/r/[code]/route.js: cookie, user-agent and
// x-forwarded-for (Go's verify rate limiter keys on IP+code).

const CODE_RE = /^[A-Za-z0-9_-]{1,30}$/;

function clientForward(req) {
  const headers = new Headers();
  for (const name of [
    "user-agent",
    "referer",
    "cookie",
    "accept",
    "accept-language",
    "content-type",
  ]) {
    const v = req.headers.get(name);
    if (v) headers.set(name, v);
  }
  const peer = req.headers.get("x-forwarded-for") || req.ip || req.headers.get("x-real-ip");
  if (peer) headers.set("x-forwarded-for", peer);
  return headers;
}

export async function POST(req, props) {
  const params = await props.params;
  const code = params?.code || "";
  if (!CODE_RE.test(code)) {
    return new NextResponse("Kode link tidak valid", {
      status: 400,
      headers: { "Content-Type": "text/plain; charset=utf-8" },
    });
  }

  let body;
  try {
    body = await req.text();
  } catch {
    return new NextResponse("Invalid request body", {
      status: 400,
      headers: { "Content-Type": "text/plain; charset=utf-8" },
    });
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/r/${code}/verify`, {
      method: "POST",
      headers: clientForward(req),
      body,
      redirect: "manual",
      cache: "no-store",
    });
  } catch {
    return new NextResponse("Go API tidak terjangkau", {
      status: 502,
      headers: { "Content-Type": "text/plain; charset=utf-8" },
    });
  }

  const out = new Headers();
  for (const name of ["location", "content-type", "cache-control", "set-cookie"]) {
    const v = goRes.headers.get(name);
    if (v) out.set(name, v);
  }
  const text = await goRes.text();
  return new NextResponse(text, { status: goRes.status, headers: out });
}
