import { NextResponse } from "next/server";
import { goErrorPayload } from "../../../lib/goError";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Same-origin proxy -> Go API so the browser never triggers CORS.
// The browser only talks to Next; Next forwards to Go server-side.
// The session cookie is forwarded as well so links created by a logged-in user
// are automatically assigned to their creator (without it, every dashboard
// link would be anonymous).
export async function POST(req) {
  let body;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/shorten`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Cookie: cookie },
      body: JSON.stringify(body),
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const text = await goRes.text();
  if (!goRes.ok) {
    // Go answers API failures as {code,message} JSON (backend/internal/
    // apierror): forward the code so the browser can translate by it.
    return NextResponse.json(goErrorPayload(text, "Go API error"), { status: goRes.status });
  }
  return NextResponse.json({ shortUrl: text }, { status: 201 });
}
