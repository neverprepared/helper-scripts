package pim

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// GetAccessToken shells out to `az account get-access-token --resource <scope>`
// and returns the bearer token. The token is a JWT for the requested resource.
func GetAccessToken(scope string) (string, error) {
	cmd := exec.Command("az", "account", "get-access-token",
		"--resource", scope,
		"--query", "accessToken",
		"-o", "tsv",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("az get-access-token (%s): %s", scope, msg)
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("az returned empty token for scope %s", scope)
	}
	return token, nil
}

// GetUserInfo parses the `oid` and `unique_name` claims out of a JWT access
// token. No signature verification — we only need the subject identifier.
// PIM activation requests use the OID as PrincipalId / SubjectId.
func GetUserInfo(jwtToken string) (AzureUserInfo, error) {
	parts := strings.Split(jwtToken, ".")
	if len(parts) < 2 {
		return AzureUserInfo{}, errors.New("not a JWT: missing claims segment")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some Azure-issued JWTs have padding; try standard URL encoding too.
		raw, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return AzureUserInfo{}, fmt.Errorf("decode JWT claims: %w", err)
		}
	}
	var info AzureUserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return AzureUserInfo{}, fmt.Errorf("parse JWT claims: %w", err)
	}
	if info.ObjectID == "" {
		return AzureUserInfo{}, errors.New("JWT missing oid claim")
	}
	return info, nil
}
