import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy timeseries -> Go GET /api/analytics/timeseries?range=. Unlike the
// clicks-by-day proxy (the OLD endpoint, returning a plain array): this
// endpoint answers {range, data_per, days}, so the range and the read source
// travel along to the chart on the Analytics tab.
export async function GET(req) {
  const cookie = req.headers.get("cookie") || "";
  const qs = new URL(req.url).search || "";

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/api/analytics/timeseries${qs}`, {
      headers: { Cookie: cookie },
      cache: "no-store",
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
