// goErrorPayload: normalize one upstream Go error body for the browser.
//
// Go answers API failures as {"code","message"} JSON (see
// backend/internal/apierror), but proxies historically forwarded the raw
// body as {error: text}. Parsing here keeps the browser contract stable:
// {error: <human message>, code: <stable key?>}. Non-JSON upstream bodies
// (legacy plaintext, proxy fallbacks) pass through as {error: text}.
export function goErrorPayload(text, fallback) {
  try {
    const g = JSON.parse(text);
    if (g && typeof g === "object" && typeof g.code === "string" && g.code !== "") {
      return { error: g.message || text, code: g.code };
    }
  } catch {
    // Not JSON: legacy plaintext upstream - keep the old shape.
  }
  return { error: text || fallback };
}
