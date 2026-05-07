package azprofile

import (
	"fmt"
)

const ablyMessageName = "snapshot"

// ChannelName builds the per-(identity, profile) channel name.
// Format: <prefix>.<sha256(upn)[:12]>.<profile>
func ChannelName(prefix, upn, profile string) string {
	if prefix == "" {
		prefix = "azprofile"
	}
	if profile == "" {
		profile = "default"
	}
	return fmt.Sprintf("%s.%s.%s", prefix, AzureUPNHash(upn), profile)
}
