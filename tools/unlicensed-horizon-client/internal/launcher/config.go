package launcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const Project = "unlicensed-horizon-client"

var nickRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)

type Config struct {
	Nickname       string `json:"nickname"`
	ClientPath     string `json:"client_path"`
	ResourcesPath  string `json:"resources_path"`
	HomePath       string `json:"home_path"`
	World          string `json:"world"`
	Server         string `json:"server,omitempty"`
	RenderDistance int    `json:"render_distance"`
	Creative       bool   `json:"creative"`
}

func DataDir() (string, error) {
	if v := os.Getenv("UHC_DATA_DIR"); v != "" {
		return filepath.Abs(v)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, Project), nil
}
func DefaultConfig(dataDir string) Config {
	home := filepath.Join(dataDir, "game-home")
	resources := ""
	if runtime.GOOS == "windows" {
		if app := os.Getenv("APPDATA"); app != "" {
			candidate := filepath.Join(app, "horizon", "game-files", "horizon-client", "26.3")
			if existsDir(candidate) {
				resources = candidate
			}
		}
	}
	return Config{Nickname: "Player", HomePath: home, RenderDistance: 12, ResourcesPath: resources}
}
func configFile(dataDir string) string { return filepath.Join(dataDir, "config.json") }
func Load(dataDir string) (Config, error) {
	c := DefaultConfig(dataDir)
	b, err := os.ReadFile(configFile(dataDir))
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("config parse: %w", err)
	}
	return c, nil
}
func Save(dataDir string, c Config) error {
	if err := Validate(c); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := configFile(dataDir)
	f, err := os.CreateTemp(dataDir, "config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Rename replaces an existing file on POSIX but not on Windows. Preserve the
	// previous config as a backup and restore it if the replacement fails.
	if err := os.Rename(f.Name(), path); err != nil {
		if !existsFile(path) {
			return err
		}
		backup, e := os.CreateTemp(dataDir, "config-backup-*.json")
		if e != nil {
			return e
		}
		backupPath := backup.Name()
		_ = backup.Close()
		_ = os.Remove(backupPath)
		if e = os.Rename(path, backupPath); e != nil {
			return fmt.Errorf("preserve existing config: %w", e)
		}
		if e = os.Rename(f.Name(), path); e != nil {
			if restoreErr := os.Rename(backupPath, path); restoreErr != nil {
				return fmt.Errorf("save failed: %v; old config remains at %s (restore: %v)", e, backupPath, restoreErr)
			}
			return e
		}
		_ = os.Remove(backupPath)
	}
	return nil
}
func Validate(c Config) error {
	if !nickRe.MatchString(c.Nickname) {
		return errors.New("nickname must have 3-16 ASCII letters, digits or underscores")
	}
	if c.RenderDistance < 2 || c.RenderDistance > 128 {
		return errors.New("render distance must be 2..128")
	}
	if c.Server != "" {
		if err := ValidateServer(c.Server); err != nil {
			return err
		}
		if c.World != "" {
			return errors.New("choose either a world or a server, not both")
		}
	}
	if len(c.World) > 64 {
		return errors.New("world name too long (max 64 characters)")
	}
	if c.World != "" {
		if c.World == "." || c.World == ".." || strings.TrimSpace(c.World) != c.World || strings.HasSuffix(c.World, ".") {
			return errors.New("world name is not a safe Windows folder name")
		}
		for _, r := range c.World {
			if r < 32 || strings.ContainsRune(`/\:<>|?*"`, r) {
				return errors.New("world name contains invalid characters")
			}
		}
		stem := strings.ToUpper(strings.Split(c.World, ".")[0])
		for _, reserved := range []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"} {
			if stem == reserved {
				return errors.New("world name is a reserved Windows filename")
			}
		}
	}
	return nil
}
func existsDir(s string) bool  { info, err := os.Stat(s); return err == nil && info.IsDir() }
func existsFile(s string) bool { info, err := os.Stat(s); return err == nil && info.Mode().IsRegular() }
func ClientExecutableName() string {
	if runtime.GOOS == "windows" {
		return "horizon-client.exe"
	}
	return "horizon-client"
}

// ValidateServer accepts a DNS name, IPv4 address or bracketed IPv6 address,
// each with an optional numeric port. It never accepts schemes, URLs, credentials
// or paths. Offline profile names are impersonable; use private trusted servers.
func ValidateServer(raw string) error {
	if len(raw) > 255 || strings.TrimSpace(raw) != raw || strings.ContainsAny(raw, "/\\@?# \t\n\r") {
		return errors.New("server must be HOST[:PORT] (no URL or spaces)")
	}
	host, port := raw, ""
	if strings.HasPrefix(raw, "[") {
		if strings.HasSuffix(raw, "]") {
			host = strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")
		} else {
			h, p, err := net.SplitHostPort(raw)
			if err != nil {
				return errors.New("invalid bracketed IPv6 server address")
			}
			host, port = h, p
		}
		if ip := net.ParseIP(host); ip == nil || !strings.Contains(host, ":") {
			return errors.New("brackets are only allowed for IPv6")
		}
	} else {
		if strings.Count(raw, ":") > 1 {
			return errors.New("IPv6 server addresses must use [IPv6]:PORT")
		}
		if h, p, ok := strings.Cut(raw, ":"); ok {
			host, port = h, p
		}
		if host == "" || len(host) > 253 {
			return errors.New("invalid server host")
		}
		if net.ParseIP(host) == nil {
			for _, label := range strings.Split(host, ".") {
				if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
					return errors.New("invalid server host label")
				}
				for _, ch := range label {
					if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-') {
						return errors.New("invalid server host characters")
					}
				}
			}
		}
	}
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("server port must be 1..65535")
		}
	}
	if strings.HasSuffix(raw, ":") {
		return errors.New("missing server port")
	}
	return nil
}

func PreparedResources(path string) error {
	report := AuditResources(path)
	if len(report.Issues) > 0 {
		return errors.New(strings.Join(report.Issues, "; "))
	}
	return nil
}
func Check(c Config) []string {
	issues := []string{}
	if err := Validate(c); err != nil {
		issues = append(issues, err.Error())
	}
	if c.ClientPath == "" || !existsFile(c.ClientPath) {
		issues = append(issues, "engine executable missing: use Install Engine or select horizon-client.exe")
	}
	if err := PreparedResources(c.ResourcesPath); err != nil {
		issues = append(issues, err.Error())
	}
	return issues
}
