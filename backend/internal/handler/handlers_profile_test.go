package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"jejak/internal/auth"
	"jejak/internal/db"
)

func TestHandleUpdateProfileTheme(t *testing.T) {
	s := &fakeStore{creator: db.Creator{ID: 1, Username: "test", Theme: "classic"}}
	h := newTestHandler(s)
	h.Auth = auth.NewMemoryStore()

	// Valid preset "coral" → 200 and echoed theme "coral".
	req := authedPut(s, h, 1, "/api/profile", `{"display_name":"Test","bio":"","avatar_url":"","socials":[],"theme":"coral"}`)
	rr := httptest.NewRecorder()
	h.HandleUpdateMyProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("coral: status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["theme"] != "coral" {
		t.Errorf("echo theme = %v, want coral", resp["theme"])
	}

	// "darkroom" → 200.
	req2 := authedPut(s, h, 1, "/api/profile", `{"display_name":"Test","bio":"","avatar_url":"","socials":[],"theme":"darkroom"}`)
	rr2 := httptest.NewRecorder()
	h.HandleUpdateMyProfile(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("darkroom: status = %d, want 200; body = %s", rr2.Code, rr2.Body.String())
	}

	// Preset eksperimen "glass" (preset 4) → 200.
	for _, preset := range []string{"glass"} {
		reqX := authedPut(s, h, 1, "/api/profile", `{"display_name":"Test","bio":"","avatar_url":"","socials":[],"theme":"`+preset+`"}`)
		rrX := httptest.NewRecorder()
		h.HandleUpdateMyProfile(rrX, reqX)
		if rrX.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200; body = %s", preset, rrX.Code, rrX.Body.String())
		}
	}

	// Legacy "night" is rejected on write (closed set = classic|darkroom|coral|glass).
	req5 := authedPut(s, h, 1, "/api/profile", `{"display_name":"Test","bio":"","avatar_url":"","socials":[],"theme":"night"}`)
	rr5 := httptest.NewRecorder()
	h.HandleUpdateMyProfile(rr5, req5)
	if rr5.Code != http.StatusBadRequest {
		t.Fatalf("legacy night on write: status = %d, want 400; body = %s", rr5.Code, rr5.Body.String())
	}

	// Invalid preset → 400.
	req3 := authedPut(s, h, 1, "/api/profile", `{"display_name":"Test","bio":"","avatar_url":"","socials":[],"theme":"neon"}`)
	rr3 := httptest.NewRecorder()
	h.HandleUpdateMyProfile(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Fatalf("neon: status = %d, want 400; body = %s", rr3.Code, rr3.Body.String())
	}

	// GET returns validTheme(creator.Theme) — unknown coerces to "classic".
	s.creator.Theme = ""
	req4 := authedReq(s, h, 1, "/api/profile", "")
	req4.Method = http.MethodGet
	rr4 := httptest.NewRecorder()
	h.HandleGetMyProfile(rr4, req4)
	var prof map[string]any
	json.Unmarshal(rr4.Body.Bytes(), &prof)
	if prof["theme"] != "classic" {
		t.Errorf("empty theme coerced to = %v, want classic", prof["theme"])
	}

	// Legacy "night" row on GET maps to "darkroom" (rebranding, bukan classic).
	s.creator.Theme = "night"
	rr6 := httptest.NewRecorder()
	h.HandleGetMyProfile(rr6, authedReq(s, h, 1, "/api/profile", ""))
	var prof6 map[string]any
	json.Unmarshal(rr6.Body.Bytes(), &prof6)
	if prof6["theme"] != "darkroom" {
		t.Errorf("legacy night mapped to = %v, want darkroom", prof6["theme"])
	}
}

func TestHandleUploadAvatar(t *testing.T) {
	// Fixture magic bytes — HEADER format ASLI (bukan cuma ekstensi).
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00}
	png := append(
		[]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'},
		[]byte{0x00, 0x00, 0x00, 0x0D, 'I', 'H', 'D', 'R', 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01}...,
	)
	// PHP shell yang berpenampilan .jpg — ancaman nyata, magic bytes WAJIB menolak.
	phpShell := []byte("<?php system($_GET['c']); ?>")

	t.Run("anonymous rejected", func(t *testing.T) {
		s := &fakeStore{creator: db.Creator{ID: 42, DisplayName: "T", Theme: "classic"}}
		h := newTestHandler(s)
		req := authedMultipart(t, h, 0, "avatar", "photo.jpg", jpeg)
		rr := httptest.NewRecorder()
		h.HandleUploadAvatar(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous status = %d, want 401", rr.Code)
		}
		if len(s.profileCalls) != 0 {
			t.Fatalf("store touched for anonymous upload: %+v", s.profileCalls)
		}
	})

	t.Run("valid jpeg saved with extension from magic bytes", func(t *testing.T) {
		s := &fakeStore{creator: db.Creator{ID: 777001, DisplayName: "T", Theme: "classic"}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		// Nama file sengaja BOHONG (.png) — ekstensi harus tetap .jpg dari magic bytes.
		req := authedMultipart(t, h, 777001, "avatar", "fake-name.png", jpeg)
		rr := httptest.NewRecorder()
		h.HandleUploadAvatar(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("jpeg status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		var resp map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("response not JSON: %v", err)
		}
		if resp["avatar_url"] != "/uploads/avatars/777001.jpg" {
			t.Errorf("avatar_url = %q, want /uploads/avatars/777001.jpg", resp["avatar_url"])
		}
		if len(s.profileCalls) != 1 || s.profileCalls[0].avatarURL != "/uploads/avatars/777001.jpg" {
			t.Errorf("UpdateCreatorProfile not persisted as jpg: %+v", s.profileCalls)
		}
		disk := filepath.Join("uploads", "avatars", "777001.jpg")
		defer os.RemoveAll(disk)
		f, err := os.Open(disk)
		if err != nil {
			t.Fatalf("saved file not found: %v", err)
		}
		defer f.Close()
		got := make([]byte, len(jpeg))
		if _, err := io.ReadFull(f, got); err != nil {
			t.Fatalf("read saved file: %v", err)
		}
		if !bytes.Equal(got, jpeg) {
			t.Error("saved bytes != uploaded bytes")
		}
	})

	t.Run("valid png saved as .png", func(t *testing.T) {
		s := &fakeStore{creator: db.Creator{ID: 777002, DisplayName: "T", Theme: "classic"}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := authedMultipart(t, h, 777002, "avatar", "real.png", png)
		rr := httptest.NewRecorder()
		h.HandleUploadAvatar(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("png status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		if len(s.profileCalls) != 1 || s.profileCalls[0].avatarURL != "/uploads/avatars/777002.png" {
			t.Errorf("png avatar not persisted: %+v", s.profileCalls)
		}
		defer os.RemoveAll(filepath.Join("uploads", "avatars", "777002.png"))
	})

	t.Run("php shell with .jpg name rejected by magic bytes", func(t *testing.T) {
		s := &fakeStore{creator: db.Creator{ID: 42, DisplayName: "T", Theme: "classic"}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := authedMultipart(t, h, 42, "avatar", "photo.jpg", phpShell)
		rr := httptest.NewRecorder()
		h.HandleUploadAvatar(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("shell bypassed magic bytes: status = %d, want 400; body=%s", rr.Code, rr.Body.String())
		}
		if len(s.profileCalls) != 0 {
			t.Fatalf("store written for rejected upload: %+v", s.profileCalls)
		}
	})

	t.Run("missing file field", func(t *testing.T) {
		s := &fakeStore{creator: db.Creator{ID: 42, DisplayName: "T", Theme: "classic"}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := authedMultipart(t, h, 42, "notavatar", "photo.jpg", jpeg)
		rr := httptest.NewRecorder()
		h.HandleUploadAvatar(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("missing field status = %d, want 400", rr.Code)
		}
	})

	t.Run("oversize rejected before write", func(t *testing.T) {
		s := &fakeStore{creator: db.Creator{ID: 42, DisplayName: "T", Theme: "classic"}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		big := make([]byte, 3<<20)
		copy(big, jpeg)
		req := authedMultipart(t, h, 42, "avatar", "big.jpg", big)
		rr := httptest.NewRecorder()
		h.HandleUploadAvatar(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("oversize status = %d, want 400; body=%s", rr.Code, rr.Body.String())
		}
		if len(s.profileCalls) != 0 {
			t.Fatalf("store written for oversize upload: %+v", s.profileCalls)
		}
		if _, err := os.Stat(filepath.Join("uploads", "avatars", "42.jpg")); err == nil {
			t.Fatal("oversize file written to disk; want no file")
		}
	})
}
