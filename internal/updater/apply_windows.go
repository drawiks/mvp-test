//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// ApplyAndRestart stages the downloaded binary next to the running executable,
// swaps it in place and launches the new process.
func ApplyAndRestart(newPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}

	staged := exe + ".new"
	old := exe + ".old"
	_ = os.Remove(staged)
	if err := copyFile(newPath, staged); err != nil {
		return fmt.Errorf("prepare update: %w", err)
	}
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		_ = os.Remove(staged)
		return fmt.Errorf("replace update: %w", err)
	}
	if err := os.Rename(staged, exe); err != nil {
		_ = os.Rename(old, exe)
		return fmt.Errorf("replace update: %w", err)
	}

	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	if err := cmd.Start(); err != nil {
		_ = os.Remove(exe)
		_ = os.Rename(old, exe)
		return fmt.Errorf("restart: %w", err)
	}
	return nil
}

// CleanupOld removes files left behind by a previous update.
func CleanupOld() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = os.Remove(exe + ".old")
	_ = os.Remove(exe + ".new")
	if matches, err := filepath.Glob(filepath.Join(os.TempDir(), "mvp-update-*")); err == nil {
		for _, path := range matches {
			_ = os.RemoveAll(path)
		}
	}
}
