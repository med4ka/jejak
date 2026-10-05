// translateError: render one API failure in the visitor's language.
//
// The Go API answers failures as {"code","message"} (see
// backend/internal/apierror): code is the stable SCREAMING_SNAKE key,
// message the Indonesian fallback for clients without dictionaries.
// Proxies forward {error, code} (see lib/goError.js), so err may carry
// either `message` or `error` as the human text.
//
//   translateError({code: "LINK_NOT_FOUND", message: "..."}, t)
//     -> t("errors.LINK_NOT_FOUND") when the dictionary has it,
//        else the message, else `fallback`, else errors.UNKNOWN.
//
// A missing dictionary entry returns the KEY ITSELF (translate() rule),
// which is why the lookup explicitly rejects that case instead of showing
// "errors.LINK_NOT_FOUND" to the user.
export function translateError(err, t, fallback) {
  const code = err && typeof err === "object" ? err.code : undefined;
  if (typeof code === "string" && code !== "") {
    const key = "errors." + code;
    const s = t(key);
    if (typeof s === "string" && s !== "" && s !== key) return s;
  }
  if (err && typeof err === "object" && (err.message || err.error)) {
    return err.message || err.error;
  }
  if (typeof err === "string" && err !== "") return err;
  if (typeof fallback === "string" && fallback !== "") return fallback;
  return t("errors.UNKNOWN");
}
