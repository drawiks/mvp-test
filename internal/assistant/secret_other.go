//go:build !windows

package assistant

import "encoding/base64"

// EncryptSecret is a development fallback. Release builds are Windows-only and
// use DPAPI in secret_windows.go.
func EncryptSecret(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return base64.StdEncoding.EncodeToString([]byte(plain)), nil
}

func DecryptSecret(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
