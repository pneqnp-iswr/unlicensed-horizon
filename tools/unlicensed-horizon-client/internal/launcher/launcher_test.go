package launcher

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// mockPreparedPreflight is NOT a playable Horizon data set. It only satisfies
// the documented static preflight for mock-process unit tests.
func mockPreparedPreflight(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := InitOpenLab(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, LabMarker)); err != nil {
		t.Fatal(err)
	}
	// Mock preflight fixture is intentionally NOT the lab; strip the lab marker.
	if err := os.WriteFile(filepath.Join(dir, "horizon-resources.json"), []byte(`{"data_version":5023}`), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestProfile(t *testing.T) {
	cfg := DefaultConfig(t.TempDir())
	cfg.Nickname = "Rafig_123"
	cfg.World = "New World"
	cfg.RenderDistance = 16
	cfg.Creative = true
	cfg.ResourcesPath = "C:/some where/resources"
	expected := []string{"--name", "Rafig_123", "--resources", "C:/some where/resources", "--render-distance", "16", "--world", "New World", "--creative"}
	if got := Arguments(cfg); !reflect.DeepEqual(got, expected) {
		t.Fatalf("args: %#v", got)
	}
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "ab", "name with spaces", "aaaaaaaaaaaaaaaaa", "../evil"} {
		copy := cfg
		copy.Nickname = bad
		if err := Validate(copy); err == nil {
			t.Errorf("accepted bad name %q", bad)
		}
	}
	cfg.World = "../../bad"
	if Validate(cfg) == nil {
		t.Error("accepted path traversal world")
	}
}
func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)
	cfg.Nickname = "abc_123"
	cfg.ClientPath = "/tmp/horizon client"
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, got) {
		t.Errorf("saved=%+v, loaded=%+v", cfg, got)
	}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
}
func TestCheck(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)
	cfg.ClientPath = filepath.Join(dir, "horizon-client.exe")
	cfg.ResourcesPath = filepath.Join(dir, "data")
	if err := os.Mkdir(cfg.ResourcesPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.ClientPath, []byte("fake"), 0700); err != nil {
		t.Fatal(err)
	}
	if len(Check(cfg)) == 0 {
		t.Fatal("accepted empty resources directory")
	}
	mockPreparedPreflight(t, cfg.ResourcesPath)
	if issues := Check(cfg); len(issues) != 0 {
		t.Errorf("unexpected issues: %v", issues)
	}
}
func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, data := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestSafeUnpack(t *testing.T) {
	for _, name := range []string{"../attack", "/abs", "C:bad.exe", "foo\\bar"} {
		dir := t.TempDir()
		if err := extractSafeZIP(zipBytes(t, map[string]string{name: "test"}), dir); err == nil {
			t.Errorf("accepted malicious zip entry %q", name)
		}
	}
	dir := t.TempDir()
	if err := extractSafeZIP(zipBytes(t, map[string]string{"bundle/horizon-client.exe": "binary"}), dir); err != nil {
		t.Fatal(err)
	}
	exe, err := findInTree(dir, "horizon-client.exe")
	if err != nil || !strings.HasSuffix(exe, "horizon-client.exe") {
		t.Errorf("not found: %q %v", exe, err)
	}
}
func TestReleaseSelection(t *testing.T) {
	releases := []GHRelease{
		{Tag: "launcher-v1", Assets: []GHAsset{{Name: "horizon-windows-x64.zip"}, {Name: "SHA256SUMS"}, {Name: "SHA256SUMS.sig"}}},
		{Tag: "nightly-x", Assets: []GHAsset{{Name: "horizon-windows-x64.zip"}, {Name: "SHA256SUMS"}, {Name: "SHA256SUMS.sig"}}},
	}
	r, _, _, err := chooseRelease(releases, "horizon-windows-x64.zip")
	if err != nil || r.Tag != "nightly-x" {
		t.Errorf("unexpected release: %+v %v", r, err)
	}
	sum, err := checksumFor([]byte(strings.Repeat("a", 64)+"  horizon-windows-x64.zip\n"), "horizon-windows-x64.zip")
	if err != nil || sum != strings.Repeat("a", 64) {
		t.Errorf("unexpected checksum %q %v", sum, err)
	}
}

func TestLaunchLocalMock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX shell mock")
	}
	dir := t.TempDir()
	cfg := DefaultConfig(dir)
	cfg.Nickname = "Local_123"
	cfg.ResourcesPath = filepath.Join(dir, "resources")
	cfg.ClientPath = filepath.Join(dir, "mock-horizon-client")
	cfg.HomePath = filepath.Join(dir, "home")
	if err := os.MkdirAll(cfg.ResourcesPath, 0755); err != nil {
		t.Fatal(err)
	}
	mockPreparedPreflight(t, cfg.ResourcesPath)
	script := "#!/bin/sh\nprintf '%s\\n' \"$HORIZON_HOME\" > \"$HORIZON_HOME/home-path.txt\"\npwd > \"$HORIZON_HOME/cwd.txt\"\nprintf '%s\\n' \"$@\" > \"$HORIZON_HOME/args.txt\"\n"
	if err := os.WriteFile(cfg.ClientPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	pid, err := Launch(cfg)
	if err != nil || pid <= 0 {
		t.Fatalf("launch failed: %d %v", pid, err)
	}
	end := time.Now().Add(3 * time.Second)
	var content []byte
	for time.Now().Before(end) {
		content, err = os.ReadFile(filepath.Join(cfg.HomePath, "args.txt"))
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Local_123") {
		t.Errorf("unexpected args: %q", content)
	}
	home, err := os.ReadFile(filepath.Join(cfg.HomePath, "home-path.txt"))
	if err != nil || strings.TrimSpace(string(home)) != cfg.HomePath {
		t.Errorf("wrong HORIZON_HOME: %q %v", home, err)
	}
	cwd, err := os.ReadFile(filepath.Join(cfg.HomePath, "cwd.txt"))
	if err != nil || strings.TrimSpace(string(cwd)) != cfg.HomePath {
		t.Errorf("game current directory wrong: %q %v", cwd, err)
	}
}

func TestReleaseSignedVerificationSynthetic(t *testing.T) {
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sums := []byte(strings.Repeat("a", 64) + "  horizon-windows-x64.zip\n")
	for _, tag := range []string{"nightly", "nightly-2026-10-29", "v2026.10.29"} {
		signTag := tag
		if strings.HasPrefix(tag, "nightly") {
			signTag = "nightly"
		}
		sig := ed25519.Sign(private, append([]byte("horizon release "+signTag+"\n"), sums...))
		b64 := []byte(base64.StdEncoding.EncodeToString(sig) + "\n")
		if err := verifyReleaseWithKey(tag, sums, b64, pub); err != nil {
			t.Errorf("verify %s: %v", tag, err)
		}
		if err := verifyReleaseWithKey(tag, append([]byte(nil), sums[:len(sums)-1]...), b64, pub); err == nil {
			t.Error("accepted changed checksums")
		}
		if err := verifyReleaseWithKey("vwrong", sums, b64, pub); err == nil {
			t.Error("accepted wrong release tag")
		}
	}
	if err := VerifyReleaseSignature("../evil", sums, nil); err == nil {
		t.Error("invalid tag accepted")
	}
	if err := VerifyReleaseSignature("nightly", sums, nil); err == nil {
		t.Error("unsigned manifest accepted")
	}
}
func TestWorldAndDistanceValidation(t *testing.T) {
	cfg := DefaultConfig(t.TempDir())
	for _, bad := range []string{"..", ".", "CON", "AUX.txt", "name.", "name ", "a<b", "new|world"} {
		cfg.World = bad
		if err := Validate(cfg); err == nil {
			t.Errorf("accepted world %q", bad)
		}
	}
	cfg.World = "Survival"
	cfg.RenderDistance = 128
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.RenderDistance = 129
	if err := Validate(cfg); err == nil {
		t.Error("accepted render distance 129")
	}
}

func TestFindResources(t *testing.T) {
	app := t.TempDir()
	t.Setenv("APPDATA", app)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	dir := filepath.Join(app, "horizon", "game-files", "horizon-client", "26.3")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := FindResources(); err == nil {
		t.Fatal("accepted unprepared folder")
	}
	mockPreparedPreflight(t, dir)
	found, err := FindResources()
	if err != nil || found != dir {
		t.Errorf("got %q (%v), want %q", found, err, dir)
	}
}

func TestWaitAndProcessLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX shell mock")
	}
	dir := t.TempDir()
	cfg := DefaultConfig(dir)
	cfg.ClientPath = filepath.Join(dir, "mock-horizon")
	cfg.ResourcesPath = filepath.Join(dir, "prepared")
	cfg.HomePath = filepath.Join(dir, "home")
	if err := os.MkdirAll(cfg.ResourcesPath, 0755); err != nil {
		t.Fatal(err)
	}
	mockPreparedPreflight(t, cfg.ResourcesPath)
	script := "#!/bin/sh\necho 'mock startup' >&2\nexit 7\n"
	if err := os.WriteFile(cfg.ClientPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := LaunchWait(cfg); err == nil {
		t.Fatal("error exit was not propagated")
	}
	b, err := os.ReadFile(ProcessLog(cfg))
	if err != nil || !strings.Contains(string(b), "mock startup") {
		t.Fatalf("missing game stderr: %q %v", b, err)
	}
}

func TestServersAndResourcesAudit(t *testing.T) {
	cfg := DefaultConfig(t.TempDir())
	for _, address := range []string{"localhost", "127.0.0.1:25565", "example.com:25565", "[::1]:25565", "[2001:db8::1]"} {
		cfg.Server = address
		if err := Validate(cfg); err != nil {
			t.Errorf("good server %q: %v", address, err)
		}
		args := Arguments(cfg)
		if !strings.Contains(strings.Join(args, " "), "--server "+address) {
			t.Error("missing server argument")
		}
	}
	for _, address := range []string{"http://evil.invalid", "evil/a", "[::1", "example.com:0", "example.com:99999", "host:abc", "-bad.example", "not @host", "2001:db8::1", "a..b", ""} {
		if address == "" {
			continue
		}
		cfg.Server = address
		if err := Validate(cfg); err == nil {
			t.Errorf("accepted bad server %q", address)
		}
	}
	cfg.World = "myworld"
	cfg.Server = "localhost"
	if err := Validate(cfg); err == nil {
		t.Error("accepted world and server together")
	}
	dir := t.TempDir()
	a := AuditResources(dir)
	if a.ReadyForAttempt || len(a.Issues) < 7 {
		t.Fatalf("accepted empty resources: %+v", a)
	}
	mockPreparedPreflight(t, dir)
	a = AuditResources(dir)
	if !a.ReadyForAttempt {
		t.Fatalf("rejected synthetic preflight fixture: %+v", a)
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.json"), []byte("not JSON"), 0644); err != nil {
		t.Fatal(err)
	}
	if AuditResources(dir).ReadyForAttempt {
		t.Error("accepted invalid facts JSON")
	}
	if err := os.WriteFile(filepath.Join(dir, "horizon-resources.json"), []byte(`{"data_version":0}`), 0644); err != nil {
		t.Fatal(err)
	}
	if AuditResources(dir).ReadyForAttempt {
		t.Error("accepted wrong data version")
	}
}

func TestLegacyWorldDiscovery(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)
	cfg.ClientPath = filepath.Join(dir, "engines", "old", "horizon-client")
	source := filepath.Join(filepath.Dir(cfg.ClientPath), "saves")
	world := filepath.Join(source, "World One")
	if err := os.MkdirAll(world, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(world, "level.dat"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	foundPath, worlds, err := LegacyWorlds(cfg)
	if err != nil || foundPath != source || len(worlds) != 1 || worlds[0] != "World One" {
		t.Fatalf("legacy discovery: %s, %v, %v", foundPath, worlds, err)
	}
}

func TestOpenContentLabGeneratedFiles(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "original-open-lab")
	root, err := InitOpenLab(dest)
	if err != nil || root != dest {
		t.Fatalf("lab init: %s, %v", root, err)
	}
	manifest, issues := AuditOpenLab(dest)
	if len(issues) > 0 || len(manifest.GeneratedFiles) < 30 {
		t.Fatalf("lab audit issues: %v, files=%d", issues, len(manifest.GeneratedFiles))
	}
	if manifest.ReadyForGame || !manifest.Experimental {
		t.Fatal("lab must never claim playable")
	}
	a := AuditResources(dest)
	if a.ReadyForAttempt || !strings.Contains(strings.Join(a.Issues, ";"), "experimental") {
		t.Fatalf("lab incorrectly passed doctor: %+v", a)
	}
	if _, err := InitOpenLab(dest); err == nil {
		t.Fatal("overwrote existing lab")
	}
	file := filepath.Join(dest, "facts.json")
	if err := os.WriteFile(file, []byte(`{"states":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, issues = AuditOpenLab(dest)
	if !strings.Contains(strings.Join(issues, ";"), "hash mismatch facts.json") {
		t.Fatalf("failed to catch tampering: %v", issues)
	}
}
func TestOpenLabDeterministic(t *testing.T) {
	one := filepath.Join(t.TempDir(), "one")
	two := filepath.Join(t.TempDir(), "two")
	if _, err := InitOpenLab(one); err != nil {
		t.Fatal(err)
	}
	if _, err := InitOpenLab(two); err != nil {
		t.Fatal(err)
	}
	a, _ := AuditOpenLab(one)
	b, _ := AuditOpenLab(two)
	if !reflect.DeepEqual(a.GeneratedFiles, b.GeneratedFiles) {
		t.Fatal("lab generation is not deterministic")
	}
}

func TestOpenLabLoaderContracts(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lab")
	if _, err := InitOpenLab(root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var facts struct {
		States   [][]json.RawMessage `json:"states"`
		Defaults map[string]int      `json:"defaults"`
	}
	if err := json.Unmarshal(raw, &facts); err != nil {
		t.Fatal(err)
	}
	if len(facts.States) != 19 || len(facts.Defaults) != 19 {
		t.Fatalf("wrong registry rows: %d/%d", len(facts.States), len(facts.Defaults))
	}
	for i, row := range facts.States {
		if len(row) < 5 {
			t.Fatalf("short facts row %d", i)
		}
		var id string
		if len(row[4]) > 0 && row[4][0] == '{' {
			var state struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(row[4], &state); err != nil {
				t.Fatal(err)
			}
			id = state.ID
		} else if err := json.Unmarshal(row[4], &id); err != nil {
			t.Fatal(err)
		}
		if facts.Defaults[id] != i {
			t.Fatalf("bad defaults for %s: %d != %d", id, facts.Defaults[id], i)
		}
	}
	b, err := os.ReadFile(filepath.Join(root, "block-tags.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tags map[string]string
	if err := json.Unmarshal(b, &tags); err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"minecraft:blocks_motion_in_heightmap", "minecraft:blocks_motion_in_heightmap_no_leaves", "minecraft:blocks_fluid_flow"} {
		var v struct {
			Values []string `json:"values"`
		}
		if err := json.Unmarshal([]byte(tags[tag]), &v); err != nil || v.Values == nil {
			t.Fatalf("invalid tag %s: %v", tag, err)
		}
	}
	f, err := os.Open(filepath.Join(root, "assets", "minecraft", "textures", "colormap", "grass.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := png.DecodeConfig(f)
	if err != nil || info.Width != 256 || info.Height != 256 {
		t.Fatalf("bad colormap: %+v %v", info, err)
	}
}

// A matching SHA-256 inventory alone is insufficient: a mutated file could be
// re-hashed by whoever edits the lab. Contract checks reject broken semantics.
func TestOpenLabRejectsBrokenCrossReferences(t *testing.T) {
	edits := []struct {
		name, file, replacement, expected string
	}{
		{"bad-default-index", "facts.json", `"minecraft:stone": 0`, "default state index"},
		{"unresolved-tag", "block-tags.json", `minecraft:stone`, "nonexistent block"},
		{"missing-variant", "assets/minecraft/blockstates/stone.json", `minecraft:block/stone`, "unexpected model reference"},
		{"bad-model-texture", "assets/minecraft/models/block/stone.json", `minecraft:block/stone`, "broken model-to-texture"},
	}
	for _, tc := range edits {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "lab")
			if _, err := InitOpenLab(root); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, filepath.FromSlash(tc.file))
			b, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			change := []byte{}
			switch tc.name {
			case "bad-default-index":
				change = bytes.Replace(b, []byte(tc.replacement), []byte(`"minecraft:stone": 0`), 1)
				// Json.Marshal sorts keys and includes double spacing? Mutate decoded JSON to be robust.
				var doc map[string]any
				if err := json.Unmarshal(b, &doc); err != nil {
					t.Fatal(err)
				}
				doc["defaults"].(map[string]any)["minecraft:stone"] = float64(0)
				change, _ = json.Marshal(doc)
			case "unresolved-tag":
				var doc map[string]string
				if err := json.Unmarshal(b, &doc); err != nil {
					t.Fatal(err)
				}
				doc["minecraft:blocks_motion_in_heightmap"] = `{"values":["minecraft:not_a_block"]}`
				change, _ = json.Marshal(doc)
			case "missing-variant":
				change = bytes.ReplaceAll(b, []byte(tc.replacement), []byte("minecraft:block/nonexistent"))
			case "bad-model-texture":
				change = bytes.ReplaceAll(b, []byte(tc.replacement), []byte("minecraft:block/missing"))
			}
			if err := os.WriteFile(target, change, 0644); err != nil {
				t.Fatal(err)
			}
			got := CheckLabContract(root)
			if got.SubsetConsistent || !strings.Contains(strings.Join(got.Issues, ";"), tc.expected) {
				t.Fatalf("expected %s: %+v", tc.expected, got)
			}
			// Even a forged checksum must not bypass the structural validation.
			manifestFile := filepath.Join(root, LabMarker)
			manifestBytes, err := os.ReadFile(manifestFile)
			if err != nil {
				t.Fatal(err)
			}
			var m LabManifest
			if err := json.Unmarshal(manifestBytes, &m); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(change)
			m.GeneratedFiles[tc.file] = hex.EncodeToString(sum[:])
			rewritten, _ := json.Marshal(m)
			if err := os.WriteFile(manifestFile, rewritten, 0644); err != nil {
				t.Fatal(err)
			}
			_, issues := AuditOpenLab(root)
			if !strings.Contains(strings.Join(issues, ";"), "contract: ") {
				t.Fatalf("forged checksum bypassed contract check: %v", issues)
			}
		})
	}
}

func TestOpenLabRejectsCorruptPNGAndDetachedMarker(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lab")
	if _, err := InitOpenLab(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "model-states.json")); err != nil {
		t.Fatal(err)
	}
	// Removing only the lab manifest is not sufficient to mark it playable.
	if err := os.Remove(filepath.Join(root, LabMarker)); err != nil {
		t.Fatal(err)
	}
	preflight := AuditResources(root)
	if preflight.ReadyForAttempt || !strings.Contains(strings.Join(preflight.Issues, ";"), "experimental") {
		t.Fatalf("unplayable lab passed markerless preflight: %+v", preflight)
	}
	pngPath := filepath.Join(root, "assets", "minecraft", "textures", "block", "stone.png")
	if err := os.WriteFile(pngPath, []byte("fake png"), 0644); err != nil {
		t.Fatal(err)
	}
	report := CheckLabContract(root)
	if report.SubsetConsistent || !strings.Contains(strings.Join(report.Issues, ";"), "invalid PNG") {
		t.Fatalf("missed corrupt PNG: %+v", report)
	}
}

func TestReadContainedFileRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "text.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readContainedFile(root, "link/text.json", 1024); err == nil {
		t.Fatal("followed symlink outside lab")
	}
	if _, err := readContainedFile(root, "../text.json", 1024); err == nil {
		t.Fatal("accepted parent traversal")
	}
}
