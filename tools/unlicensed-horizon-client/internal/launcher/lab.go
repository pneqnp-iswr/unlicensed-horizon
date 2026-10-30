package launcher

// The open-content laboratory is an ORIGINAL, intentionally incomplete dataset.
// It demonstrates the first real Horizon registry and texture contracts; it is
// NOT a Minecraft resource replacement and cannot boot a playable game.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const LabMarker = "open-content-lab.json"
const HorizonDataVersion = 5023 // 26.3: upstream crates/horizon-core/src/version.rs

var labMaterials = []string{"bedrock", "dirt", "stone", "sand", "oak_planks", "bricks"}

// Names and properties taken from the Horizon inspection gallery requirements.
// All state definitions and all pixels below are independently authored here.
type galleryState struct {
	Name       string
	Properties map[string]string
}

var labGallery = []galleryState{
	{"oak_stairs", map[string]string{"facing": "north", "half": "top", "shape": "inner_left"}},
	{"stone_slab", map[string]string{"type": "top"}},
	{"anvil", map[string]string{"facing": "east"}},
	{"torch", nil},
	{"lantern", nil},
	{"oak_fence", map[string]string{"north": "true", "east": "true"}},
	{"oak_fence_gate", map[string]string{"facing": "west", "open": "true"}},
	{"iron_bars", map[string]string{"north": "true", "south": "true"}},
	{"cobweb", nil},
	{"grindstone", map[string]string{"face": "wall", "facing": "west"}},
	{"ladder", map[string]string{"facing": "west"}},
	{"amethyst_cluster", map[string]string{"facing": "east"}},
}

func (g galleryState) PropertiesOrEmpty() map[string]string {
	if g.Properties == nil {
		return map[string]string{}
	}
	return g.Properties
}

var labFluidTextures = []string{"water_still", "water_flow", "water_overlay", "lava_still", "lava_flow"}

type LabManifest struct {
	Format         string            `json:"format"`
	DataVersion    int               `json:"data_version"`
	Experimental   bool              `json:"experimental"`
	ReadyForGame   bool              `json:"ready_for_game"`
	Description    string            `json:"description"`
	GeneratedFiles map[string]string `json:"generated_files_sha256"`
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
func writeLab(root, relative string, bytes []byte) error {
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}
func makeTile(seed int, size int, col color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Geometric, procedurally authored pattern, not a copy of game assets.
			wave := (x*3 + y*5 + seed*11 + (x*y)%17) % 19
			shift := int(wave) - 9
			shiftChan := func(v uint8, amount int) uint8 {
				q := int(v) + amount
				if q < 0 {
					q = 0
				}
				if q > 255 {
					q = 255
				}
				return uint8(q)
			}
			img.SetRGBA(x, y, color.RGBA{shiftChan(col.R, shift*2), shiftChan(col.G, shift*2), shiftChan(col.B, shift*2), 255})
		}
	}
	return img
}
func pngBytes(img image.Image) ([]byte, error) {
	var b bufferWriter
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.bytes, nil
}

type bufferWriter struct{ bytes []byte }

func (b *bufferWriter) Write(p []byte) (int, error) {
	b.bytes = append(b.bytes, p...)
	return len(p), nil
}

// InitOpenLab refuses to overwrite an existing destination. Every generated
// file is listed by its SHA-256 hash in the lab manifest for provenance.
func InitOpenLab(destination string) (string, error) {
	if destination == "" {
		return "", errors.New("empty lab destination")
	}
	root, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if _, err = os.Lstat(root); err == nil {
		return "", fmt.Errorf("destination already exists; refusing overwrite: %s", root)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err = os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(root)
		}
	}()
	files := map[string]string{}
	put := func(relative string, b []byte) error {
		if err := writeLab(root, relative, b); err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		files[relative] = hex.EncodeToString(sum[:])
		return nil
	}
	putJSON := func(relative string, value any) error {
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		return put(relative, append(b, '\n'))
	}
	// Horizon's first startup check and Assets::load data-version gate.
	manifest := map[string]any{"data_version": HorizonDataVersion, "source": "UHC original-content experiment", "content_status": "incomplete: not playable"}
	if err = putJSON("horizon-resources.json", manifest); err != nil {
		return "", err
	}
	// The exact five-column facts schema is used by horizon-world/registry/facts.rs.
	rows := [][]any{{"minecraft:air", true, false, "Air", "minecraft:air"}}
	defaults := map[string]int{"minecraft:air": 0}
	modelStates := map[string]any{}
	palette := []color.RGBA{{65, 73, 91, 255}, {126, 93, 70, 255}, {132, 143, 163, 255}, {219, 183, 112, 255}, {164, 112, 75, 255}, {156, 84, 86, 255}}
	// A miniature gallery-compatible block-state set; models are intentionally
	// generic cubes. Minecraft's actual geometry/textures are NOT supplied.
	specimens := make([]galleryState, 0, len(labMaterials)+len(labGallery))
	for _, name := range labMaterials {
		specimens = append(specimens, galleryState{Name: name})
	}
	specimens = append(specimens, labGallery...)
	for i, spec := range specimens {
		key := "minecraft:" + spec.Name
		state := any(key)
		if len(spec.Properties) > 0 {
			state = map[string]any{"id": key, "properties": spec.Properties}
		}
		defaults[key] = i + 1
		rows = append(rows, []any{key, false, false, "Original UHC experiment", state})
		colorChoice := palette[i%len(palette)]
		tex, e := pngBytes(makeTile(i+1, 16, colorChoice))
		if e != nil {
			return "", e
		}
		if err = put("assets/minecraft/textures/block/"+spec.Name+".png", tex); err != nil {
			return "", err
		}
		variant := ""
		schema := map[string]any{}
		if len(spec.Properties) > 0 {
			keys := make([]string, 0, len(spec.Properties))
			for k := range spec.Properties {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, k+"="+spec.Properties[k])
				schema[k] = map[string][]string{"named": {spec.Properties[k]}}
			}
			variant = strings.Join(parts, ",")
		}
		if err = putJSON("assets/minecraft/blockstates/"+spec.Name+".json", map[string]any{"variants": map[string]any{variant: map[string]any{"model": "minecraft:block/" + spec.Name}}}); err != nil {
			return "", err
		}
		if err = putJSON("assets/minecraft/models/block/"+spec.Name+".json", map[string]any{"parent": "minecraft:block/cube_all", "textures": map[string]string{"all": "minecraft:block/" + spec.Name, "particle": "minecraft:block/" + spec.Name}}); err != nil {
			return "", err
		}
		modelStates[key] = map[string]any{"schema": schema, "states": []map[string]string{spec.PropertiesOrEmpty()}}
	}
	if err = putJSON("model-states.json", modelStates); err != nil {
		return "", err
	}
	// Author the six-face cube parent ourselves, so models do not depend on an
	// unbundled vanilla cube_all.json. Texture references resolve per child.
	faces := map[string]any{}
	for _, face := range []string{"north", "south", "east", "west", "up", "down"} {
		faces[face] = map[string]string{"texture": "#all", "cullface": face}
	}
	cube := map[string]any{"elements": []any{map[string]any{"from": []int{0, 0, 0}, "to": []int{16, 16, 16}, "faces": faces}}}
	if err = putJSON("assets/minecraft/models/block/cube_all.json", cube); err != nil {
		return "", err
	}
	if err = putJSON("facts.json", map[string]any{"states": rows, "defaults": defaults}); err != nil {
		return "", err
	}
	// Horizon expects the value of each tag bundle entry to be a *JSON string*.
	tags := map[string]string{
		"minecraft:blocks_motion_in_heightmap":           `{"values":["minecraft:bedrock","minecraft:dirt","minecraft:stone","minecraft:sand","minecraft:oak_planks","minecraft:bricks"]}`,
		"minecraft:blocks_motion_in_heightmap_no_leaves": `{"values":["minecraft:bedrock","minecraft:dirt","minecraft:stone","minecraft:sand","minecraft:oak_planks","minecraft:bricks"]}`,
		"minecraft:blocks_fluid_flow":                    `{"values":[]}`,
	}
	if err = putJSON("block-tags.json", tags); err != nil {
		return "", err
	}
	if err = putJSON("block-behaviours.json", map[string]any{}); err != nil {
		return "", err
	}
	for i, name := range append(append([]string{}, labFluidTextures...), func() []string {
		v := []string{}
		for j := 0; j < 10; j++ {
			v = append(v, fmt.Sprintf("destroy_stage_%d", j))
		}
		return v
	}()...) {
		tile, e := pngBytes(makeTile(50+i, 16, color.RGBA{70, 126, 177, 255}))
		if e != nil {
			return "", e
		}
		if err = put("assets/minecraft/textures/block/"+name+".png", tile); err != nil {
			return "", err
		}
	}
	for i, name := range []string{"grass", "foliage", "dry_foliage"} {
		base := []color.RGBA{{74, 151, 105, 255}, {92, 144, 93, 255}, {163, 140, 81, 255}}[i]
		tile, e := pngBytes(makeTile(90+i, 256, base))
		if e != nil {
			return "", e
		}
		if err = put("assets/minecraft/textures/colormap/"+name+".png", tile); err != nil {
			return "", err
		}
	}
	if err = putJSON("data/minecraft/worldgen/biome/plains.json", map[string]any{"temperature": 0.7, "downfall": 0.4, "has_precipitation": true, "effects": map[string]any{"water_color": 4159204}, "attributes": map[string]any{}}); err != nil {
		return "", err
	}
	// These independently-authored environment inputs are read by
	// horizon-client-render/src/environment.rs::Environment::load_dimension.
	// They are NOT copied from Mojang's vanilla timelines or dimensions.
	if err = putJSON("data/minecraft/timeline/day.json", map[string]any{
		"period_ticks": 24000,
		"tracks": map[string]any{"minecraft:visual/sky_light_factor": map[string]any{
			"ease": "linear", "keyframes": []any{
				map[string]any{"ticks": 0, "value": 1.0},
				map[string]any{"ticks": 12000, "value": 0.15},
			},
		}},
	}); err != nil {
		return "", err
	}
	if err = putJSON("data/minecraft/dimension_type/overworld.json", map[string]any{
		"attributes": map[string]any{
			"minecraft:visual/sky_light_color":     "#e4edff",
			"minecraft:visual/ambient_light_color": "#121821",
			"minecraft:visual/fog_start_distance":  0.1,
			"minecraft:visual/fog_end_distance":    1.0,
		},
	}); err != nil {
		return "", err
	}
	if err = putJSON("assets/minecraft/lang/en_us.json", map[string]string{"block.minecraft.stone": "Lab Slate", "block.minecraft.dirt": "Lab Soil"}); err != nil {
		return "", err
	}
	// Include an explicit provenance marker; doctor must never regard this as prepared vanilla data.
	lab := LabManifest{Format: "uhc-open-content-lab-v3", DataVersion: HorizonDataVersion, Experimental: true, ReadyForGame: false,
		Description: "Original assets and a gallery-backed miniature block registry; missing complete worldgen/render/gameplay tables. Do not point Play at this folder.", GeneratedFiles: files}
	if err = writeJSON(filepath.Join(root, LabMarker), lab); err != nil {
		return "", err
	}
	success = true
	return root, nil
}

// AuditOpenLab verifies both content integrity and experimental/not-playable flags.
func AuditOpenLab(root string) (LabManifest, []string) {
	issues := []string{}
	b, err := os.ReadFile(filepath.Join(root, LabMarker))
	if err != nil {
		return LabManifest{}, []string{"lab manifest missing or unreadable"}
	}
	var m LabManifest
	if err = json.Unmarshal(b, &m); err != nil {
		return m, []string{"malformed lab manifest"}
	}
	if (m.Format != "uhc-open-content-lab-v1" && m.Format != "uhc-open-content-lab-v2" && m.Format != "uhc-open-content-lab-v3") || !m.Experimental || m.ReadyForGame || m.DataVersion != HorizonDataVersion {
		issues = append(issues, "invalid lab provenance metadata")
	}
	if len(m.GeneratedFiles) < 20 {
		issues = append(issues, "incomplete lab file inventory")
	}
	paths := make([]string, 0, len(m.GeneratedFiles))
	for p := range m.GeneratedFiles {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		if filepath.IsAbs(rel) || filepath.Clean(rel) == ".." || rel == ".." || rel == "." {
			issues = append(issues, "invalid lab path "+rel)
			continue
		}
		b, e := readContainedFile(root, rel, 32<<20)
		if e != nil {
			issues = append(issues, "missing/unsafe file "+rel+": "+e.Error())
			continue
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != m.GeneratedFiles[rel] {
			issues = append(issues, "hash mismatch "+rel)
		}
	}
	if m.Format == "uhc-open-content-lab-v2" || m.Format == "uhc-open-content-lab-v3" {
		report := CheckLabContract(root)
		for _, issue := range report.Issues {
			issues = append(issues, "contract: "+issue)
		}
	}
	return m, issues
}
