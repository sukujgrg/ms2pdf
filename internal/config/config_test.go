package config

import "testing"

func TestResolveClientIDFlag(t *testing.T) {
	id, err := ResolveClientID("flag-id")
	if err != nil {
		t.Fatal(err)
	}
	if id != "flag-id" {
		t.Fatalf("got %q", id)
	}
}

func TestResolveClientIDEnv(t *testing.T) {
	t.Setenv("MS2PDF_CLIENT_ID", "env-id")
	id, err := ResolveClientID("")
	if err != nil {
		t.Fatal(err)
	}
	if id != "env-id" {
		t.Fatalf("got %q", id)
	}
}

func TestResolveTenantFlag(t *testing.T) {
	id, err := ResolveTenant("tenant-id")
	if err != nil {
		t.Fatal(err)
	}
	if id != "tenant-id" {
		t.Fatalf("got %q", id)
	}
}

func TestResolveTenantEnv(t *testing.T) {
	t.Setenv("MS2PDF_TENANT_ID", "env-tenant")
	id, err := ResolveTenant("")
	if err != nil {
		t.Fatal(err)
	}
	if id != "env-tenant" {
		t.Fatalf("got %q", id)
	}
}
