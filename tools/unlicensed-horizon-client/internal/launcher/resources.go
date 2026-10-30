package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResourceAudit is a static preflight, NEVER a full Horizon boot test.
// The early checks follow the actual 26.3 asset loader, not a guessed file list.
type ResourceAudit struct {
	Path            string         `json:"path"`
	ReadyForAttempt bool           `json:"ready_for_attempt"`
	Files           []ResourceFile `json:"files"`
	Issues          []string       `json:"issues"`
	Note            string         `json:"note"`
}
type ResourceFile struct {
	Name      string `json:"name"`
	Present   bool   `json:"present"`
	Bytes     int64  `json:"bytes"`
	JSONValid bool   `json:"json_valid"`
}

func AuditResources(path string) ResourceAudit {
	a := ResourceAudit{Path: path, Files: []ResourceFile{}, Issues: []string{}, Note: "Static preflight only: later resource tables, models and gameplay compatibility require an actual Horizon boot. The independent lab is intentionally blocked."}
	if path == "" {
		a.Issues = append(a.Issues, "prepared resource directory not configured")
		return a
	}
	if !existsDir(path) {
		a.Issues = append(a.Issues, "prepared resource directory not found: "+path)
		return a
	}
	if _, err := os.Lstat(filepath.Join(path, LabMarker)); err == nil {
		a.Issues = append(a.Issues, "experimental open-content laboratory is not a playable prepared game; do not use this for Play")
	}
	var manifest map[string]json.RawMessage
	for _, name := range []string{"horizon-resources.json", "facts.json", "block-tags.json", "block-behaviours.json", "model-states.json"} {
		f := ResourceFile{Name: name}
		p := filepath.Join(path, name)
		info, err := os.Lstat(p)
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			a.Issues = append(a.Issues, "missing or empty "+name)
			a.Files = append(a.Files, f)
			continue
		}
		f.Present = true
		f.Bytes = info.Size()
		if info.Size() > 256<<20 {
			a.Issues = append(a.Issues, "oversized JSON "+name)
			a.Files = append(a.Files, f)
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil || !json.Valid(b) || strings.TrimSpace(string(b)) == "null" {
			a.Issues = append(a.Issues, "invalid JSON in "+name)
			a.Files = append(a.Files, f)
			continue
		}
		f.JSONValid = true
		switch name {
		case "horizon-resources.json":
			if err := json.Unmarshal(b, &manifest); err != nil {
				a.Issues = append(a.Issues, "invalid resource manifest object")
			} else {
				var status string
				_ = json.Unmarshal(manifest["content_status"], &status)
				var source string
				_ = json.Unmarshal(manifest["source"], &source)
				if status != "" || strings.Contains(source, "UHC original-content") {
					a.Issues = append(a.Issues, "experimental game-data marker detected; this is not playable prepared game data")
				}
				var ver int
				if json.Unmarshal(manifest["data_version"], &ver) != nil || ver != HorizonDataVersion {
					a.Issues = append(a.Issues, fmt.Sprintf("resource data_version does not match Horizon 26.3 (%d)", HorizonDataVersion))
				}
			}
		case "facts.json":
			var entry struct {
				States   []json.RawMessage `json:"states"`
				Defaults map[string]int    `json:"defaults"`
			}
			if json.Unmarshal(b, &entry) != nil || len(entry.States) == 0 || len(entry.Defaults) == 0 {
				a.Issues = append(a.Issues, "facts.json missing states/defaults registry")
			}
		case "model-states.json":
			var models map[string]json.RawMessage
			if json.Unmarshal(b, &models) != nil || len(models) == 0 {
				a.Issues = append(a.Issues, "model-states.json missing/invalid model tables")
			}
		case "block-tags.json":
			var tags map[string]string
			if json.Unmarshal(b, &tags) != nil {
				a.Issues = append(a.Issues, "block-tags.json must be an object of JSON strings")
			} else {
				for _, key := range []string{"minecraft:blocks_motion_in_heightmap", "minecraft:blocks_motion_in_heightmap_no_leaves", "minecraft:blocks_fluid_flow"} {
					val, ok := tags[key]
					if !ok || !json.Valid([]byte(val)) {
						a.Issues = append(a.Issues, "invalid or missing tag "+key)
					}
				}
			}
		}
		a.Files = append(a.Files, f)
	}
	for _, name := range []string{"assets/minecraft/textures/colormap/grass.png", "assets/minecraft/textures/colormap/foliage.png", "assets/minecraft/textures/colormap/dry_foliage.png"} {
		f := ResourceFile{Name: name}
		stat, err := os.Lstat(filepath.Join(path, filepath.FromSlash(name)))
		if err != nil || !stat.Mode().IsRegular() || stat.Size() == 0 {
			a.Issues = append(a.Issues, "missing colormap "+name)
		} else {
			f.Present = true
			f.Bytes = stat.Size()
		}
		a.Files = append(a.Files, f)
	}
	// The biome loader scans the biome directory and expects at least one JSON.
	biomeDir := filepath.Join(path, "data", "minecraft", "worldgen", "biome")
	if entries, err := os.ReadDir(biomeDir); err != nil {
		a.Issues = append(a.Issues, "missing biome folder: data/minecraft/worldgen/biome")
	} else {
		found := false
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				found = true
				break
			}
		}
		if !found {
			a.Issues = append(a.Issues, "no biome JSON found")
		}
	}
	a.ReadyForAttempt = len(a.Issues) == 0
	return a
}
