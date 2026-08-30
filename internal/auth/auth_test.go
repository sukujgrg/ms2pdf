package auth

import "testing"

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
