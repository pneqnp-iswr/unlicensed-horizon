package launcher

import (
	"os"
	"path/filepath"
	"sort"
)

// LegacyWorlds locates worlds that v0.1-v0.2 accidentally wrote alongside
// horizon-client.exe. This never moves or deletes a user save.
func LegacyWorlds(c Config) (string, []string, error) {
	if c.ClientPath == "" {
		return "", nil, nil
	}
	engine, err := filepath.Abs(c.ClientPath)
	if err != nil {
		return "", nil, err
	}
	source := filepath.Join(filepath.Dir(engine), "saves")
	entries, err := os.ReadDir(source)
	if os.IsNotExist(err) {
		return source, nil, nil
	}
	if err != nil {
		return source, nil, err
	}
	var worlds []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".horizon-title" {
			continue
		}
		if _, err := os.Stat(filepath.Join(source, entry.Name(), "level.dat")); err == nil {
			worlds = append(worlds, entry.Name())
		}
	}
	sort.Strings(worlds)
	return source, worlds, nil
}
