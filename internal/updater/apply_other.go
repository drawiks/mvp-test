//go:build !windows

package updater

import "errors"

// ApplyAndRestart is only implemented for the Windows build.
func ApplyAndRestart(string) error {
	return errors.New("автообновление поддерживается только в Windows-сборке")
}

// CleanupOld is a no-op outside Windows.
func CleanupOld() {}
