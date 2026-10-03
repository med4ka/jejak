// Link expiry time conversions (link management, 2026-09-30).
//
// Timezone policy (decision by the project owner): the backend stores and
// sends UTC (RFC3339 with Z), while the form uses LOCAL time. Conversion is
// therefore REQUIRED on both sides:
//   send    : datetime-local input (local) → toISOString()  → UTC to API
//   display : ISO UTC from API → toLocaleString('id-ID')    → local on screen
// Never show a raw ISO string to the user (hours would be misread as +7).

function pad(n) {
  return String(n).padStart(2, "0");
}

// ISO/Date (UTC) → datetime-local input value "YYYY-MM-DDTHH:MM" (local).
export function toInputValue(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// datetime-local input value (local) → ISO UTC string for the API; empty →
// null (= clears expiry on PUT; on POST simply omitting it is enough).
export function fromInputValue(v) {
  if (!v) return null;
  const d = new Date(v); // parser uses the local timezone
  if (Number.isNaN(d.getTime())) return null;
  return d.toISOString();
}

// Lower bound for the shorten input: now + 1 hour (local): the backend
// rejects anything ≤ 1 hour with 422, and the input min prevents selections
// that are guaranteed to be rejected.
export function minInputValue() {
  return toInputValue(new Date(Date.now() + 60 * 60 * 1000).toISOString());
}

// ISO UTC → local display "1 Okt 2026, 16.00" (id-ID dateStyle/timeStyle).
export function formatLocal(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" });
}

// Per-minute comparison key: second-level precision differences of the
// datetime-local input must not count as "changed" (avoids sending
// expires_at when the value is identical).
export function minuteKey(iso) {
  if (!iso) return null;
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return null;
  return Math.floor(t / 60000);
}
