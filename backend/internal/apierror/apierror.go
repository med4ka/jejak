// Package apierror is the single error envelope for the JSON API: every
// API failure answers {"code","message"} (plus "field" for per-field
// validation errors), so the frontend can translate by code instead of
// matching on Indonesian prose.
//
// Contract:
//   - code    = machine-readable SCREAMING_SNAKE, stable across locales.
//     Frontend dictionaries carry errors.<CODE> in id/en/de; unknown codes
//     fall back to message (see frontend/lib/errors.js translateError).
//   - message = human-readable Indonesian fallback. It is what a client
//     without the dictionaries shows: never empty, never a bare code.
//   - field   = only on 400/422 field errors (which input to highlight).
//
// Code catalogue (grouped; the full list with meanings):
//
//	Transport:  METHOD_NOT_ALLOWED, INVALID_REQUEST_BODY, RATE_LIMIT_EXCEEDED,
//	            LOGIN_RATE_LIMITED, REGISTER_RATE_LIMITED, DATABASE_ERROR,
//	            DATABASE_UNAVAILABLE, INTERNAL_ERROR, NOT_FOUND
//	Auth:       AUTH_NOT_CONFIGURED, AUTH_INVALID_USERNAME,
//	            AUTH_INVALID_REGISTER, AUTH_USERNAME_TAKEN,
//	            AUTH_INVALID_CREDENTIALS, AUTH_SESSION_ERROR, AUTH_HASH_ERROR,
//	            AUTH_ACCOUNT_NOT_FOUND, AUTH_WRONG_PASSWORD, AUTH_REQUIRED,
//	            AUTH_INVALID (bad API key)
//	Account:    ACCOUNT_INVALID_EMAIL, ACCOUNT_CURRENT_PASSWORD_REQUIRED,
//	            ACCOUNT_EMAIL_TAKEN, ACCOUNT_PASSWORD_REQUIRED,
//	            ACCOUNT_PASSWORD_TOO_SHORT, ACCOUNT_REVOKE_FAILED,
//	            ACCOUNT_CONFIRM_MISMATCH
//	Links:      LINK_INVALID_URL, LINK_CODE_ERROR, LINK_INVALID_SLUG,
//	            LINK_SLUG_RESERVED, LINK_SLUG_TAKEN, LINK_INVALID_TAGS,
//	            LINK_PASSWORD_HASH_ERROR, LINK_STORE_ERROR,
//	            LINK_INVALID_DEVICE_RULES, LINK_INVALID_DEVICE_URL,
//	            LINK_INVALID_CODE, LINK_REORDER_LIMIT, LINK_ORDER_REQUIRED,
//	            LINK_CLAIM_LIMIT, LINK_NOT_FOUND, LINK_NOTHING_TO_UPDATE,
//	            LINK_PASSWORD_INVALID, LINK_FALLBACK_URL_INVALID,
//	            LINK_EXPIRES_AT_INVALID, LINK_EXPIRES_AT_TOO_SOON,
//	            DEEPLINK_UNSUPPORTED, BULK_NO_VALID_URLS, BULK_URLS_INVALID,
//	            BULK_TAG_INVALID
//	Health:     HEALTH_UNAVAILABLE, HEALTH_CHECK_FAILED,
//	            NOTIFICATION_INVALID_ID, NOTIFICATION_NOT_FOUND
//	API keys:   APIKEY_NOT_FOUND, APIKEY_INVALID_ID, APIKEY_INVALID,
//	            APIKEY_LABEL_TOO_LONG, APIKEY_GENERATE_ERROR,
//	            APIKEY_STORE_ERROR
//	Profile:    PROFILE_NOT_FOUND, PROFILE_DISPLAY_NAME_REQUIRED,
//	            PROFILE_BIO_TOO_LONG, PROFILE_AVATAR_URL_INVALID,
//	            PROFILE_INVALID_THEME, PROFILE_SOCIALS_LIMIT,
//	            PROFILE_SOCIAL_PLATFORM_REQUIRED, PROFILE_SOCIAL_URL_INVALID,
//	            PROFILE_SOCIALS_INVALID
//	Avatar:     AVATAR_TOO_LARGE, AVATAR_MISSING, AVATAR_TOO_SMALL,
//	            AVATAR_BAD_TYPE
//	Analytics:  ANALYTICS_INVALID_KIND, ANALYTICS_INVALID_RANGE,
//	            ANALYTICS_INVALID_MODE
//	QR:         QR_NO_MATCH, QR_GENERATE_ERROR, QR_TOO_MANY
//	WhatsApp:   WHATSAPP_PHONE_INVALID, WHATSAPP_MESSAGE_TOO_LONG
//
// Deliberately NOT JSON (browser-facing surfaces, documented):
//   - GET /r/{code} redirect errors ("Link disabled", "URL not found"):
//     plaintext bodies for a human with a browser, no JSON consumer.
//   - HTML pages (password gate, gone/expired, health interstitial/503):
//     documents, not API responses (the password gate IS translated via
//     internal/i18n; the other static pages are a documented TODO).
package apierror

import (
	"encoding/json"
	"net/http"
)

// WriteError answers an API failure as {"code","message"} with the given
// status. It replaces http.Error throughout the JSON API: plaintext bodies
// cannot carry the machine-readable code the frontend translates by.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}

// WriteFieldError answers a per-field validation failure as
// {"code","message","field"}: the frontend highlights the named input while
// translating the message by code like any other API error.
func WriteFieldError(w http.ResponseWriter, status int, field, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message, "field": field})
}
