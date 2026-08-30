package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertFile(t *testing.T) {
	var deleted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/me/drive/root:") && strings.Contains(r.URL.Path, "/content"):
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "item1"})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "createUploadSession"):
			t.Errorf("small file should not use upload session, got %s", r.URL.Path)
			if r.Header.Get("Authorization") == "" {
				t.Error("upload session missing bearer token")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"uploadUrl": "http://" + r.Host + "/upload",
			})
		case r.Method == http.MethodPut && r.URL.Path == "/upload":
			if r.Header.Get("Authorization") != "" {
				t.Error("upload URL should not receive the Graph token")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "item1"})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/content"):
			if r.Header.Get("Authorization") == "" {
				t.Error("convert GET missing bearer token")
			}
			http.Redirect(w, r, "/pdf.bin", http.StatusFound)
		case r.URL.Path == "/pdf.bin":
			if r.Header.Get("Authorization") != "" {
				t.Error("redirect download should not receive Authorization")
			}
			_, _ = w.Write([]byte("%PDF-1.4 test"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/permanentDelete"):
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/items/item1"):
			t.Error("expected permanentDelete, got ordinary DELETE")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	in := filepath.Join(dir, "in.docx")
	out := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(in, []byte("docx-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New("tok")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	c.HTTP.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.Header.Del("Authorization")
		return nil
	}

	if err := c.ConvertFile(context.Background(), in, out, "docx"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "%PDF-1.4 test" {
		t.Fatalf("pdf body: %q", got)
	}
	if !deleted {
		t.Fatal("temp drive item was not deleted")
	}
}

func TestConvertCleanupFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/content"):
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "item1"})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/content"):
			_, _ = w.Write([]byte("%PDF-1.4 test"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/permanentDelete"):
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "accessDenied", "message": "nope"},
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	in := filepath.Join(dir, "in.docx")
	out := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(in, []byte("docx-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New("tok")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	err := c.ConvertFile(context.Background(), in, out, "docx")
	if err == nil || !strings.Contains(err.Error(), "failed to delete temp OneDrive item") {
		t.Fatalf("got %v", err)
	}
	if _, statErr := os.Stat(out); statErr != nil {
		t.Fatal("local PDF should still be written")
	}
}

func TestConvertPersonalRecycleFallback(t *testing.T) {
	var recycled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/content"):
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "item1"})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/content"):
			_, _ = w.Write([]byte("%PDF-1.4 test"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/permanentDelete"):
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "invalidRequest", "message": "API not found"},
			})
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/items/item1"):
			recycled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	in := filepath.Join(dir, "in.docx")
	out := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(in, []byte("docx-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New("tok")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	if err := c.ConvertFile(context.Background(), in, out, "docx"); err != nil {
		t.Fatal(err)
	}
	if !recycled {
		t.Fatal("personal OneDrive should fall back to DELETE")
	}
}

func TestConvertRejectsOversize(t *testing.T) {
	old := maxSourceBytes
	maxSourceBytes = 8
	t.Cleanup(func() { maxSourceBytes = old })

	dir := t.TempDir()
	in := filepath.Join(dir, "in.docx")
	if err := os.WriteFile(in, []byte("too-large"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New("tok")
	err := c.ConvertFile(context.Background(), in, filepath.Join(dir, "out.pdf"), "docx")
	if err == nil || !strings.Contains(err.Error(), "maximum for Graph PDF conversion") {
		t.Fatalf("got %v", err)
	}
}

func TestFormatProgress(t *testing.T) {
	got := formatProgress("uploading", 5<<20, 10<<20)
	if !strings.Contains(got, "uploading") || !strings.Contains(got, "50%") {
		t.Fatalf("got %q", got)
	}
	got = formatProgress("downloading", 1500, 0)
	if !strings.Contains(got, "1.5 KiB") {
		t.Fatalf("got %q", got)
	}
}

func TestOneDrivePath(t *testing.T) {
	got := oneDrivePath("abc.docx")
	if got != "OneDrive:/abc.docx" {
		t.Fatalf("got %q", got)
	}
}

func TestReporterQuietWhenNotTTY(t *testing.T) {
	var buf strings.Builder
	p := newReporter(&buf)
	p.update("uploading", 1, 2, true)
	p.line("converting…")
	p.finish()
	if buf.Len() != 0 {
		t.Fatalf("wrote %q to non-tty writer", buf.String())
	}
}

func TestReadAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "invalidRequest", "message": "nope"},
		})
	}))
	t.Cleanup(srv.Close)

	c := New("tok")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/me/drive/items/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.doGraph(req)
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("got %v", err)
	}
}
