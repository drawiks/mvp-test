//go:build windows

package assistant

import (
	"encoding/base64"
	"unsafe"

	"golang.org/x/sys/windows"
)

// EncryptSecret protects a secret with the current Windows user's DPAPI key.
func EncryptSecret(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	data := []byte(plain)
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	raw := unsafe.Slice(out.Data, out.Size)
	return base64.StdEncoding.EncodeToString(raw), nil
}

// DecryptSecret reverses EncryptSecret.
func DecryptSecret(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", nil
	}
	in := windows.DataBlob{Size: uint32(len(raw)), Data: &raw[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return string(unsafe.Slice(out.Data, out.Size)), nil
}
