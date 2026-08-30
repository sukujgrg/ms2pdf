package graph

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestCountWriterLimit(t *testing.T) {
	var buf bytes.Buffer
	w := &countWriter{w: &buf, limit: 4}
	if _, err := w.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("de")); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("got %v", err)
	}
	if buf.String() != "abc" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestCountReaderReports(t *testing.T) {
	var out bytes.Buffer
	p := &reporter{out: &out, tty: true}
	r := p.reader(strings.NewReader("hello"), 0, 5, "uploading")
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "uploading") || !strings.Contains(out.String(), "100%") {
		t.Fatalf("got %q", out.String())
	}
}
