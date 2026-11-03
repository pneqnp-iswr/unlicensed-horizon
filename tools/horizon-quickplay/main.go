//go:build windows

// Horizon QuickPlay is a small, self-contained GUI bootstrap for the
// open-source, community-maintained Horizon companion launcher.
// It contains NO Mojang/Microsoft proprietary game resources or account tokens.
package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// Generate assets/unlicensed-horizon-client.exe from the Go CLI before building.
//
//go:embed assets/unlicensed-horizon-client.exe assets/QuickPlay.ps1 assets/AutoPrepare.ps1
var bundled embed.FS

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procMessageBox = user32.NewProc("MessageBoxW")
)

func messageBox(title, body string, flags uintptr) {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(body)
	procMessageBox.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), flags)
}

func putVerifiedFile(dest string, data []byte) error {
	sum := sha256.Sum256(data)
	if current, err := os.ReadFile(dest); err == nil {
		currentSHA := sha256.Sum256(current)
		if currentSHA == sum {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(dest), ".quickplay-*")
	if err != nil {
		return err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Remove the older payload only after the new one has been created.
	if err := os.Rename(temporary, dest); err != nil {
		if removeErr := os.Remove(dest); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("cannot replace old payload: %w", removeErr)
		}
		if err := os.Rename(temporary, dest); err != nil {
			return err
		}
	}
	return nil
}

func run() error {
	base, err := os.UserCacheDir() // %LOCALAPPDATA% on Windows
	if err != nil {
		return fmt.Errorf("user cache: %w", err)
	}
	script, err := bundled.ReadFile("assets/QuickPlay.ps1")
	if err != nil {
		return err
	}
	cli, err := bundled.ReadFile("assets/unlicensed-horizon-client.exe")
	if err != nil {
		return err
	}
	preparer, err := bundled.ReadFile("assets/AutoPrepare.ps1")
	if err != nil {
		return err
	}
	hashSource := append(append(append([]byte{}, script...), preparer...), cli...)
	hash := sha256.Sum256(hashSource)
	version := hex.EncodeToString(hash[:8])
	dir := filepath.Join(base, "Horizon-QuickPlay", "payload", version)
	exe := filepath.Join(dir, "unlicensed-horizon-client.exe")
	scriptPath := filepath.Join(dir, "QuickPlay.ps1")
	preparePath := filepath.Join(dir, "AutoPrepare.ps1")
	if err := putVerifiedFile(exe, cli); err != nil {
		return err
	}
	if err := putVerifiedFile(scriptPath, script); err != nil {
		return err
	}
	if err := putVerifiedFile(preparePath, preparer); err != nil {
		return err
	}
	// Use the built-in Windows PowerShell 5.1 WinForms runtime.  The GUI is
	// visible; no PowerShell console window needs to be shown to the user.
	powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if strings.TrimSpace(os.Getenv("SystemRoot")) == "" {
		powershell = "powershell.exe"
	}
	command := exec.Command(powershell, "-NoLogo", "-NoProfile", "-NonInteractive", "-STA", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	command.Dir = dir
	command.Env = append(os.Environ(), "HORIZON_QUICKPLAY_DIR="+dir)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Run(); err != nil {
		return fmt.Errorf("cannot open QuickPlay window: %w. Ensure Windows PowerShell 5.1 is installed and enabled", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		messageBox("Horizon QuickPlay — ошибка", err.Error(), 0x10) // MB_ICONERROR
		os.Exit(1)
	}
}