package pim

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGetUserInfoExtractsOID(t *testing.T) {
	claims := `{"oid":"11111111-2222-3333-4444-555555555555","unique_name":"alice@example.com","iss":"https://sts.windows.net/..."}`
	payload := base64.RawURLEncoding.EncodeToString([]byte(claims))
	token := "header." + payload + ".signature"

	info, err := GetUserInfo(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ObjectID != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("oid = %q", info.ObjectID)
	}
	if info.Email != "alice@example.com" {
		t.Fatalf("email = %q", info.Email)
	}
}

func TestGetUserInfoRejectsMalformed(t *testing.T) {
	if _, err := GetUserInfo("not-a-jwt"); err == nil {
		t.Fatal("expected error for non-JWT input")
	}
	if _, err := GetUserInfo("header.&&&.sig"); err == nil {
		t.Fatal("expected error for un-base64 claims")
	}
	noOID := base64.RawURLEncoding.EncodeToString([]byte(`{"unique_name":"a@b"}`))
	if _, err := GetUserInfo("h." + noOID + ".s"); err == nil || !strings.Contains(err.Error(), "oid") {
		t.Fatalf("expected oid-missing error, got %v", err)
	}
}
