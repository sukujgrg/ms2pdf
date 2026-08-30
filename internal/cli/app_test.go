package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelp(t *testing.T) {
	var out bytes.Buffer
	cmd := New()
	cmd.Writer = &out
	if err := cmd.Run(context.Background(), []string{"ms2pdf", "--help"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"login", "logout", "whoami", "convert", "browser"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help missing %q:\n%s", want, got)
		}
	}
}

func TestConvertRequiresFile(t *testing.T) {
	cmd := New()
	err := cmd.Run(context.Background(), []string{"ms2pdf", "convert"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("got %v", err)
	}
}

func TestConvertRejectsCSV(t *testing.T) {
	cmd := New()
	err := cmd.Run(context.Background(), []string{"ms2pdf", "convert", "data.csv"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("got %v", err)
	}
}
