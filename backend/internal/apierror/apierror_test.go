package apierror

// Tests: the JSON error envelope shape the frontend translates by
// (translateError reads code/message/field).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusNotFound, "LINK_NOT_FOUND", "URL not found")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v (%q)", err, rr.Body.String())
	}
	if body["code"] != "LINK_NOT_FOUND" || body["message"] != "URL not found" {
		t.Fatalf("body = %v", body)
	}
	if _, ok := body["field"]; ok {
		t.Fatalf("plain error must not carry field: %v", body)
	}
}

func TestWriteFieldErrorEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteFieldError(rr, http.StatusBadRequest, "phone", "WHATSAPP_PHONE_INVALID", "phone is required")
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["code"] != "WHATSAPP_PHONE_INVALID" || body["field"] != "phone" || body["message"] == "" {
		t.Fatalf("body = %v", body)
	}
}
