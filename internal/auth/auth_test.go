package auth

import (
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

func TestAuthorityTenantID(t *testing.T) {
	got, err := Authority("93212d20-0a9b-4d19-b9ea-fa92cf33441d")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://login.microsoftonline.com/93212d20-0a9b-4d19-b9ea-fa92cf33441d"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAuthorityCommon(t *testing.T) {
	got, err := Authority("common")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://login.microsoftonline.com/common" {
		t.Fatalf("got %q", got)
	}
}

func TestAuthorityDefaultCommon(t *testing.T) {
	got, err := Authority("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://login.microsoftonline.com/common" {
		t.Fatalf("got %q", got)
	}
}

func TestDeviceCodeTenantOK(t *testing.T) {
	if DeviceCodeTenantOK("common") || DeviceCodeTenantOK("") {
		t.Fatal("aliases must be rejected")
	}
	if !DeviceCodeTenantOK("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee") {
		t.Fatal("guid must be accepted")
	}
}

func TestAccountByHomeID(t *testing.T) {
	accounts := []public.Account{
		{HomeAccountID: "aaa", PreferredUsername: "a@example.com"},
		{HomeAccountID: "bbb", PreferredUsername: "b@example.com"},
	}
	got, err := accountByHomeID(accounts, "bbb")
	if err != nil {
		t.Fatal(err)
	}
	if got.PreferredUsername != "b@example.com" {
		t.Fatalf("got %q", got.PreferredUsername)
	}
	if _, err := accountByHomeID(accounts, ""); err != ErrNotLoggedIn {
		t.Fatalf("empty id: %v", err)
	}
	if _, err := accountByHomeID(accounts, "ccc"); err != ErrNotLoggedIn {
		t.Fatalf("missing id: %v", err)
	}
	if _, err := accountByHomeID(nil, "bbb"); err != ErrNotLoggedIn {
		t.Fatalf("empty cache: %v", err)
	}
}
