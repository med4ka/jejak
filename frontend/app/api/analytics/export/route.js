import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy for the CSV download -> Go GET /api/analytics/export.csv (in-depth
// analytics). The path on the Next side omits ".csv" (/api/analytics/export)
// for tidiness; Go uses the literal "/api/analytics/export.csv". The session
// cookie + ?mode=&range= are forwarded; the UTF-8 BOM at the start of the body
// is NOT stripped (arrayBuffer intact): without the BOM, Excel reads the CSV
// as CP1252 and mangles non-ASCII characters.
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";
  const qs = new URL(req.url).search || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/analytics/export.csv${qs}`, {
      headers: { Cookie: cookie },
    });
  } catch {
    return NextResponse.json({ error: "Go API tidak terjangkau" }, { status: 502 });
  }

  const buf = await goRes.arrayBuffer();
  const headers = new Headers();
  headers.set("Content-Type", goRes.headers.get("Content-Type") || "text/csv; charset=utf-8");
  const cd = goRes.headers.get("Content-Disposition");
  if (cd) {
    headers.set("Content-Disposition", cd);
  }
  return new NextResponse(buf, { status: goRes.status, headers });
}
