package azprofile

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SyncedFiles is the minimal set of files required for `az` to authenticate
// on a receiving machine without re-running `az login`.
var SyncedFiles = []string{
	"msal_token_cache.json",
	"msal_http_cache.bin",
	"azureProfile.json",
	"clouds.config",
}

// Envelope is the plaintext payload published to Ably (then AES-GCM encrypted).
type Envelope struct {
	Version       int               `json:"v"`
	SenderID      string            `json:"sender_id"`
	TsUnixMs      int64             `json:"ts_unix_ms"`
	MonotonicSeq  int64             `json:"monotonic_seq"`
	Profile       string            `json:"profile"`
	AzureUPNHash  string            `json:"azure_upn_hash"`
	Files         map[string]string `json:"files"` // name -> base64
	ChecksumSHA256 string           `json:"checksum_sha256"`
}

// CollectFiles reads the files in SyncedFiles from profileDir.
// At minimum azureProfile.json and msal_token_cache.json must exist; the
// other two are optional (some profiles may not have them yet).
func CollectFiles(profileDir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	required := map[string]bool{
		"azureProfile.json":      true,
		"msal_token_cache.json":  true,
	}
	for _, name := range SyncedFiles {
		path := filepath.Join(profileDir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if required[name] {
					return nil, fmt.Errorf("required file missing: %s", path)
				}
				continue
			}
			return nil, err
		}
		out[name] = b
	}
	return out, nil
}

func BuildEnvelope(senderID, profile, upnHash string, seq int64, files map[string][]byte) *Envelope {
	enc := map[string]string{}
	for k, v := range files {
		enc[k] = base64.StdEncoding.EncodeToString(v)
	}
	return &Envelope{
		Version:        1,
		SenderID:       senderID,
		TsUnixMs:       time.Now().UnixMilli(),
		MonotonicSeq:   seq,
		Profile:        profile,
		AzureUPNHash:   upnHash,
		Files:          enc,
		ChecksumSHA256: filesChecksum(files),
	}
}

func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func UnmarshalEnvelope(b []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	if e.Version != 1 {
		return nil, fmt.Errorf("unsupported envelope version: %d", e.Version)
	}
	return &e, nil
}

// ApplyTo writes the envelope's files into profileDir using atomic per-file
// renames. Aborts on first error and cleans up the staging directory.
func (e *Envelope) ApplyTo(profileDir string) error {
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}
	staging := filepath.Join(profileDir, ".azprofile-incoming")
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	// Decode all first so we can validate checksum before touching profileDir.
	decoded := make(map[string][]byte, len(e.Files))
	for name, b64 := range e.Files {
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}
		decoded[name] = raw
	}
	if got, want := filesChecksum(decoded), e.ChecksumSHA256; got != want {
		return fmt.Errorf("checksum mismatch: got %s want %s", got, want)
	}

	type pending struct {
		tmp, dst string
	}
	var pendings []pending
	for name, raw := range decoded {
		dst := filepath.Join(profileDir, name)
		nonce := make([]byte, 4)
		_, _ = rand.Read(nonce)
		tmp := filepath.Join(staging, name+".tmp."+hex.EncodeToString(nonce))
		if err := os.WriteFile(tmp, raw, 0o600); err != nil {
			return err
		}
		pendings = append(pendings, pending{tmp: tmp, dst: dst})
	}
	for _, p := range pendings {
		if err := os.Rename(p.tmp, p.dst); err != nil {
			return fmt.Errorf("rename %s: %w", p.dst, err)
		}
	}
	if d, err := os.Open(profileDir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func filesChecksum(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for k := range files {
		names = append(names, k)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		h.Write([]byte(n))
		h.Write([]byte{0})
		h.Write(files[n])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
