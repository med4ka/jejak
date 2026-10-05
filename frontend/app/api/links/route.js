import { NextResponse } from "next/server";
import { goErrorPayload } from "../../../lib/goError";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Same-origin proxy -> Go API GET /api/links (links owned by the logged-in
// user; the session is forwarded so the server can scope results per creator).
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";
  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/links`, {
      cache: "no-store",
      headers: cookie ? { cookie } : {},
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const data = await goRes.json();
  if (!goRes.ok) {
    // Preserve a Go error envelope ({code,message}) when present so the
    // browser can translate by code (backend/internal/apierror).
    const code = data && typeof data.code === "string" ? data.code : undefined;
    return NextResponse.json(
      code ? { error: data.message || "Go API error", code } : { error: "Go API error" },
      { status: goRes.status }
    );
  }
  return NextResponse.json(data);
}
