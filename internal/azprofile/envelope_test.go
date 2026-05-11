package azprofile

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvelopeMarshalRoundTrip(t *testing.T) {
	files := map[string][]byte{
		"msal_token_cache.json": []byte(strings.Repeat(`{"access_token":"AAAA","refresh_token":"RRRR","scopes":["https://management.azure.com/.default"]},`, 200)),
		"azureProfile.json":     []byte(`{"subscriptions":[]}`),
	}
	e := BuildEnvelope("sender", "work", "hash", 1, files)
	wire, err := e.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !(len(wire) >= 2 && wire[0] == 0x1f && wire[1] == 0x8b) {
		t.Fatalf("expected gzip magic on wire bytes")
	}
	raw, _ := json.Marshal(e)
	if len(wire) >= len(raw) {
		t.Fatalf("expected gzip to shrink payload: raw=%d wire=%d", len(raw), len(wire))
	}
	got, err := UnmarshalEnvelope(wire)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ChecksumSHA256 != e.ChecksumSHA256 {
		t.Fatalf("checksum mismatch")
	}
}

func TestUnmarshalLegacyRawJSON(t *testing.T) {
	files := map[string][]byte{
		"msal_token_cache.json": []byte(`{"a":1}`),
		"azureProfile.json":     []byte(`{"b":2}`),
	}
	e := BuildEnvelope("sender", "work", "hash", 7, files)
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	if bytes.HasPrefix(raw, []byte{0x1f, 0x8b}) {
		t.Fatalf("raw JSON shouldn't start with gzip magic")
	}
	got, err := UnmarshalEnvelope(raw)
	if err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if got.MonotonicSeq != 7 {
		t.Fatalf("seq lost in round-trip")
	}
}

func TestSyncedFilesNoHTTPCache(t *testing.T) {
	for _, f := range SyncedFiles {
		if f == "msal_http_cache.bin" {
			t.Fatalf("msal_http_cache.bin must not be in SyncedFiles (pushes payload over Ably 64 KB limit)")
		}
	}
}
