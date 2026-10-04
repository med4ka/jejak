import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy "mark notification read" -> Go PUT /api/notifications/{id}/read.
// Session cookie forwarded; an unknown/foreign id stays 404 (no id probing).
export async function PUT(req, props) {
  const params = await props.params;
  const id = params.id;
  const cookie = req.headers.get("cookie") || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/notifications/${id}/read`, {
      method: "PUT",
      headers: { Cookie: cookie },
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const text = await goRes.text();
  return new NextResponse(text, {
    status: goRes.status,
    headers: { "Content-Type": "application/json" },
  });
}
