// Pure translation lookup shared by the client provider (I18nProvider.jsx)
// and the server helper (serverI18n.js). No React, no "use client" marker:
// safe to import from both sides.

// Dot-path lookup ("dashboard.tabs.ringkasan"); a literal top-level key
// also works, so flat dictionaries remain valid.
function resolve(messages, key) {
  if (!messages || !key) return undefined;
  if (Object.prototype.hasOwnProperty.call(messages, key)) return messages[key];
  return key
    .split(".")
    .reduce((acc, part) => (acc && typeof acc === "object" ? acc[part] : undefined), messages);
}

// "{name}" placeholders: keeps variable strings (counters, usernames,
// error detail) translatable without losing the interpolation.
function interpolate(value, vars) {
  if (!vars || typeof value !== "string") return value;
  return value.replace(/\{(\w+)\}/g, (match, name) =>
    Object.prototype.hasOwnProperty.call(vars, name) ? String(vars[name]) : match
  );
}

// Returns the translated string, or the key itself when the lookup misses
// (missing keys stay visible in the UI instead of rendering blank).
export function translate(messages, key, vars) {
  const found = resolve(messages, key);
  if (found === undefined || found === null) return key;
  return interpolate(String(found), vars);
}
