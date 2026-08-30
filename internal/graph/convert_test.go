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
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/items/item1"):
			deleted = true
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
