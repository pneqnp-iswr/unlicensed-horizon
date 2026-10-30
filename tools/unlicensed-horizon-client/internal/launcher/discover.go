package launcher

import (
	"os"
	"path/filepath"
)

// Discovery only checks well-known paths. It neither downloads proprietary
// resources nor copies them across installations.
func ResourceCandidates() []string {
	roots := []string{}
	if app := os.Getenv("APPDATA"); app != "" {
		roots = append(roots, filepath.Join(app, "horizon"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		roots = append(roots, filepath.Join(local, "Horizon"))
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		roots = append(roots, filepath.Join(xdg, "horizon"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".local", "share", "horizon"), filepath.Join(home, "AppData", "Roaming", "horizon"))
	}
	out, seen := []string{}, map[string]bool{}
	for _, root := range roots {
		candidate := filepath.Join(root, "game-files", "horizon-client", "26.3")
		if !seen[candidate] {
			seen[candidate] = true
			out = append(out, candidate)
		}
	}
	return out
}

func FindResources() (string, error) {
	for _, path := range ResourceCandidates() {
		if err := PreparedResources(path); err == nil {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}
