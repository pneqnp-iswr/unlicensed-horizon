package launcher

// This validator independently checks the subset of Horizon's *published*
// resource contracts used in our open-content laboratory. It is NOT the Rust
// loader; a passing result never claims that Horizon successfully booted.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ContractReport struct {
	Path              string   `json:"path"`
	SubsetConsistent  bool     `json:"subset_consistent"`
	RegistryStates    int      `json:"registry_states"`
	RegistryBlocks    int      `json:"registry_blocks"`
	ResolvedTags      int      `json:"resolved_tags"`
	ModelBindings     int      `json:"model_bindings"`
	PNGTextures       int      `json:"png_textures"`
	Issues            []string `json:"issues"`
	NextKnownBlockers []string `json:"next_known_blockers"`
	Note              string   `json:"note"`
}

// readContainedFile forbids symlinks in *every* path component, not just the
// final file, and enforces a fixed size bound before decoding untrusted input.
func readContainedFile(root, relative string, max int64) ([]byte, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "\\") ||
		filepath.ToSlash(filepath.Clean(relative)) != relative || relative == ".." || strings.HasPrefix(relative, "../") || strings.Contains(relative, ":") {
		return nil, fmt.Errorf("unsafe relative path %q", relative)
	}
	r, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(r)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("root not a physical directory")
	}
	parts := strings.Split(relative, "/")
	node := r
	for i, part := range parts {
		if part == "." || part == ".." || part == "" {
			return nil, fmt.Errorf("unsafe path segment")
		}
		node = filepath.Join(node, part)
		st, e := os.Lstat(node)
		if e != nil {
			return nil, e
		}
		if i == len(parts)-1 {
			if !st.Mode().IsRegular() || st.Size() > max {
				return nil, fmt.Errorf("not a regular file or too large: %s", relative)
			}
		} else if !st.IsDir() {
			return nil, fmt.Errorf("non-directory or symlink component: %s", part)
		}
	}
	return os.ReadFile(node)
}

type labState struct {
	Name       string
	Properties map[string]string
}
type labModelSpec struct {
	Schema map[string]struct {
		Named []string `json:"named"`
	} `json:"schema"`
	States []map[string]string `json:"states"`
}

func CheckLabContract(root string) ContractReport {
	r := ContractReport{Path: root, Issues: []string{}, NextKnownBlockers: []string{
		"Horizon UI resources and panorama", "Full gameplay and worldgen registries, including NBT tables", "Lighting, collision and motion tables", "Native Rust-loader boot, renderer and world-save integration tests",
	}, Note: "Only a Go structural subset checker, not a Rust Horizon loader test. NOT a playable Minecraft distribution."}
	fail := func(format string, args ...any) { r.Issues = append(r.Issues, fmt.Sprintf(format, args...)) }
	read := func(name string) ([]byte, bool) {
		b, e := readContainedFile(root, name, 32<<20)
		if e != nil {
			fail("%s: %v", name, e)
			return nil, false
		}
		return b, true
	}
	var marker map[string]json.RawMessage
	if b, ok := read("horizon-resources.json"); ok {
		if e := json.Unmarshal(b, &marker); e != nil {
			fail("invalid resource marker: %v", e)
		} else {
			var version int
			if json.Unmarshal(marker["data_version"], &version) != nil || version != HorizonDataVersion {
				fail("data_version is not %d", HorizonDataVersion)
			}
		}
	}
	if bytes, ok := read(LabMarker); ok {
		var meta LabManifest
		if json.Unmarshal(bytes, &meta) != nil || (meta.Format != "uhc-open-content-lab-v2" && meta.Format != "uhc-open-content-lab-v3") || !meta.Experimental || meta.ReadyForGame || meta.DataVersion != HorizonDataVersion {
			fail("invalid or unsupported open-content laboratory provenance")
		}
	}
	rowsByName := map[string][]labState{}
	defaults := map[string]int{}
	states := []labState{}
	if b, ok := read("facts.json"); ok {
		var doc struct {
			States   [][]json.RawMessage `json:"states"`
			Defaults map[string]int      `json:"defaults"`
		}
		if e := json.Unmarshal(b, &doc); e != nil {
			fail("invalid facts.json: %v", e)
		} else {
			defaults = doc.Defaults
			for i, row := range doc.States {
				if len(row) < 5 {
					fail("facts row %d has fewer than 5 fields", i)
					continue
				}
				var name string
				var air, fluid bool
				if json.Unmarshal(row[0], &name) != nil || json.Unmarshal(row[1], &air) != nil || json.Unmarshal(row[2], &fluid) != nil {
					fail("facts row %d has invalid name or flags", i)
					continue
				}
				state := labState{Name: name, Properties: map[string]string{}}
				if len(row[4]) == 0 {
					fail("facts row %d has empty state", i)
					continue
				}
				if row[4][0] == '"' {
					if json.Unmarshal(row[4], &state.Name) != nil {
						fail("facts row %d: invalid state name", i)
						continue
					}
				} else {
					var o struct {
						ID         string            `json:"id"`
						Properties map[string]string `json:"properties"`
					}
					if json.Unmarshal(row[4], &o) != nil || o.ID == "" {
						fail("facts row %d: invalid compound state", i)
						continue
					}
					state.Name = o.ID
					if o.Properties != nil {
						state.Properties = o.Properties
					}
				}
				if state.Name != name {
					fail("facts row %d name differs from state id", i)
				}
				if (name == "minecraft:air") != air || fluid {
					fail("facts row %d has inconsistent air/fluid flags", i)
				}
				states = append(states, state)
				rowsByName[name] = append(rowsByName[name], state)
			}
			for name, index := range defaults {
				if index < 0 || index >= len(states) || states[index].Name != name {
					fail("default state index does not reference %s", name)
				}
			}
			for name := range rowsByName {
				if _, ok := defaults[name]; !ok {
					fail("no default state for %s", name)
				}
			}
		}
	}
	// Environment loader's concrete on-disk inputs. V3 of the original lab
	// adds them; V2 remains readable for backward-compatible audit of old labs.
	if b, ok := read(LabMarker); ok {
		var m LabManifest
		if json.Unmarshal(b, &m) == nil && m.Format == "uhc-open-content-lab-v3" {
			if b, ok = read("data/minecraft/timeline/day.json"); ok {
				var timeline struct {
					Period int `json:"period_ticks"`
					Tracks map[string]struct {
						Keyframes []struct {
							Ticks int             `json:"ticks"`
							Value json.RawMessage `json:"value"`
						} `json:"keyframes"`
					} `json:"tracks"`
				}
				if json.Unmarshal(b, &timeline) != nil || timeline.Period <= 0 || timeline.Tracks == nil {
					fail("invalid day timeline")
				} else {
					for name, track := range timeline.Tracks {
						if len(track.Keyframes) == 0 {
							fail("empty timeline keyframes for %s", name)
						}
						for _, k := range track.Keyframes {
							if k.Ticks < 0 || k.Ticks >= timeline.Period || len(k.Value) == 0 {
								fail("invalid timeline keyframe for %s", name)
							}
						}
					}
				}
			}
			if b, ok = read("data/minecraft/dimension_type/overworld.json"); ok {
				var dim struct {
					Attributes map[string]json.RawMessage `json:"attributes"`
				}
				if json.Unmarshal(b, &dim) != nil || dim.Attributes == nil {
					fail("invalid original overworld dimension attributes")
				}
			}
		}
	}
	r.RegistryStates = len(states)
	r.RegistryBlocks = len(rowsByName)
	requiredTags := []string{"minecraft:blocks_motion_in_heightmap", "minecraft:blocks_motion_in_heightmap_no_leaves", "minecraft:blocks_fluid_flow"}
	type tagEnt struct {
		ID       string
		Required bool
	}
	tagFiles := map[string][]tagEnt{}
	if b, ok := read("block-tags.json"); ok {
		var bundle map[string]string
		if e := json.Unmarshal(b, &bundle); e != nil {
			fail("block-tags.json: expected object of JSON strings")
		} else {
			for name, raw := range bundle {
				var file struct {
					Values []json.RawMessage `json:"values"`
				}
				if e := json.Unmarshal([]byte(raw), &file); e != nil || file.Values == nil {
					fail("tag %s has invalid values array", name)
					continue
				}
				tagFiles[name] = []tagEnt{}
				for _, v := range file.Values {
					ent := tagEnt{Required: true}
					if len(v) > 0 && v[0] == '"' {
						_ = json.Unmarshal(v, &ent.ID)
					} else {
						var record struct {
							ID       string `json:"id"`
							Required *bool  `json:"required"`
						}
						if json.Unmarshal(v, &record) != nil {
							fail("invalid entry in tag %s", name)
							continue
						}
						ent.ID = record.ID
						if record.Required != nil {
							ent.Required = *record.Required
						}
					}
					if ent.ID == "" {
						fail("empty entry in tag %s", name)
						continue
					}
					if !strings.Contains(strings.TrimPrefix(ent.ID, "#"), ":") {
						ent.ID = strings.Replace(ent.ID, "#", "#minecraft:", 1)
						if !strings.HasPrefix(ent.ID, "#") {
							ent.ID = "minecraft:" + ent.ID
						}
					}
					tagFiles[name] = append(tagFiles[name], ent)
				}
			}
			for _, name := range requiredTags {
				if _, ok := tagFiles[name]; !ok {
					fail("missing required tag %s", name)
				}
			}
			visiting := map[string]bool{}
			visited := map[string]bool{}
			var visit func(string)
			visit = func(name string) {
				if visited[name] {
					return
				}
				if visiting[name] {
					fail("cyclic block tag: %s", name)
					return
				}
				visiting[name] = true
				for _, ent := range tagFiles[name] {
					if strings.HasPrefix(ent.ID, "#") {
						next := strings.TrimPrefix(ent.ID, "#")
						if _, ok := tagFiles[next]; !ok {
							if ent.Required {
								fail("tag %s requires missing tag %s", name, next)
							}
						} else {
							visit(next)
						}
					} else if _, ok := defaults[ent.ID]; !ok && ent.Required {
						fail("tag %s references nonexistent block %s", name, ent.ID)
					}
				}
				visiting[name] = false
				visited[name] = true
			}
			for name := range tagFiles {
				visit(name)
			}
			r.ResolvedTags = len(tagFiles)
		}
	}
	if b, ok := read("model-states.json"); ok {
		specs := map[string]labModelSpec{}
		if e := json.Unmarshal(b, &specs); e != nil {
			fail("invalid model-states.json: %v", e)
		} else {
			for name, group := range rowsByName {
				if name == "minecraft:air" {
					continue
				}
				spec, ok := specs[name]
				if !ok {
					fail("model-states missing block %s", name)
					continue
				}
				if len(spec.States) != len(group) {
					fail("model-states %s has wrong state count", name)
					continue
				}
				for i, st := range group {
					if !equalProps(st.Properties, spec.States[i]) {
						fail("model-states %s property mismatch for state %d", name, i)
					}
					for k, v := range st.Properties {
						sch, ok := spec.Schema[k]
						if !ok || !contains(sch.Named, v) {
							fail("invalid model property schema %s.%s", name, k)
						}
					}
				}
				id := strings.TrimPrefix(name, "minecraft:")
				b, ok := read("assets/minecraft/blockstates/" + id + ".json")
				if !ok {
					continue
				}
				var blockstate struct {
					Variants map[string]struct {
						Model string `json:"model"`
					} `json:"variants"`
				}
				if json.Unmarshal(b, &blockstate) != nil {
					fail("invalid blockstate for %s", name)
					continue
				}
				for _, state := range group {
					keys := make([]string, 0, len(state.Properties))
					for k := range state.Properties {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					parts := make([]string, 0, len(keys))
					for _, k := range keys {
						parts = append(parts, k+"="+state.Properties[k])
					}
					variant := strings.Join(parts, ",")
					choice, ok := blockstate.Variants[variant]
					if !ok {
						fail("missing variant %s of %s", variant, name)
						continue
					}
					if choice.Model != "minecraft:block/"+id {
						fail("unexpected model reference in %s", name)
					}
				}
				model, ok := read("assets/minecraft/models/block/" + id + ".json")
				if !ok {
					continue
				}
				var doc struct {
					Parent   string            `json:"parent"`
					Textures map[string]string `json:"textures"`
				}
				if json.Unmarshal(model, &doc) != nil || doc.Parent != "minecraft:block/cube_all" || doc.Textures["all"] != "minecraft:block/"+id {
					fail("broken model-to-texture reference in %s", name)
				}
				r.ModelBindings++
			}
		}
	}
	if b, ok := read("assets/minecraft/models/block/cube_all.json"); ok {
		var cube struct {
			Elements []json.RawMessage `json:"elements"`
		}
		if json.Unmarshal(b, &cube) != nil || len(cube.Elements) == 0 {
			fail("cube_all parent missing elements")
		}
	}
	for _, name := range []string{"grass", "foliage", "dry_foliage"} {
		checkLabPNG(root, "assets/minecraft/textures/colormap/"+name+".png", 256, 256, &r)
	}
	names := make([]string, 0, len(rowsByName))
	for name := range rowsByName {
		if name != "minecraft:air" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		id := strings.TrimPrefix(name, "minecraft:")
		checkLabPNG(root, "assets/minecraft/textures/block/"+id+".png", 16, 16, &r)
	}
	for _, name := range labFluidTextures {
		checkLabPNG(root, "assets/minecraft/textures/block/"+name+".png", 16, 16, &r)
	}
	for i := 0; i < 10; i++ {
		checkLabPNG(root, fmt.Sprintf("assets/minecraft/textures/block/destroy_stage_%d.png", i), 16, 16, &r)
	}
	r.SubsetConsistent = len(r.Issues) == 0
	return r
}
func equalProps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
func checkLabPNG(root, name string, w, h int, r *ContractReport) {
	b, err := readContainedFile(root, name, 8<<20)
	if err != nil {
		r.Issues = append(r.Issues, fmt.Sprintf("%s: %v", name, err))
		return
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		r.Issues = append(r.Issues, fmt.Sprintf("%s: invalid PNG: %v", name, err))
		return
	}
	if cfg.Width != w || cfg.Height != h {
		r.Issues = append(r.Issues, fmt.Sprintf("%s: dimension mismatch, expected %dx%d", name, w, h))
		return
	}
	// Read full image to catch truncated data that still has a valid IHDR.
	if _, err = png.Decode(bytes.NewReader(b)); err != nil {
		r.Issues = append(r.Issues, fmt.Sprintf("%s: corrupt PNG pixels", name))
		return
	}
	r.PNGTextures++
}
