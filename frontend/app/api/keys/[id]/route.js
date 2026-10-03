import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy API key deletion -> Go DELETE /api/keys/{id}. The session cookie is
// forwarded; Go verifies key ownership (a key not owned returns 404).
export async function DELETE(req, { params }) {
  const cookie = req.headers.get("cookie") || "";
  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/keys/${params.id}`, {
      method: "DELETE",
      headers: { Cookie: cookie },
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