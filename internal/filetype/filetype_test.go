package filetype

import (
	"path/filepath"
	"testing"
)

func TestResolveFromExtension(t *testing.T) {
	ext, err := Resolve("report.DOCX", "")
	if err != nil {
		t.Fatal(err)
	}
	if ext != "docx" {
		t.Fatalf("got %q", ext)
	}
}

func TestResolveTypeOverride(t *testing.T) {
	ext, err := Resolve("memo", "docx")
	if err != nil {
		t.Fatal(err)
	}
	if ext != "docx" {
		t.Fatalf("got %q", ext)
	}
}

func TestResolveMissing(t *testing.T) {
	if _, err := Resolve("memo", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveUnsupported(t *testing.T) {
	if _, err := Resolve("data.csv", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultOutput(t *testing.T) {
	got := DefaultOutput(filepath.Join("in", "report.docx"))
	want := filepath.Join("in", "report.pdf")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
