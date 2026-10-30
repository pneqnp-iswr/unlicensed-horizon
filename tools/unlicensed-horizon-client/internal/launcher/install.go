package launcher

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	// Horizon's published release key, pinned from crates/horizon-release-sign/release-keys.txt.
	// Updating this key requires a reviewed source change, not a network download.
	horizonReleaseKey = "MU4oiIQcCxkZvbwSjjXs7SySCuM6ywRQ3nGaJYZG0gw="
	apiURL            = "https://api.github.com/repos/horizon-rust/horizon/releases?per_page=30"
	maxArchive        = int64(180 << 20)
	maxExtract        = int64(950 << 20)
)

type GHAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}
type GHRelease struct {
	Tag        string    `json:"tag_name"`
	Assets     []GHAsset `json:"assets"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
}

var releaseTagRE = regexp.MustCompile(`^[a-zA-Z0-9_.@-]+$`)

func VerifyReleaseSignature(tag string, sums, signatures []byte) error {
	key, err := base64.StdEncoding.DecodeString(horizonReleaseKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("invalid bundled release key")
	}
	return verifyReleaseWithKey(tag, sums, signatures, ed25519.PublicKey(key))
}
func verifyReleaseWithKey(tag string, sums, signatures []byte, key ed25519.PublicKey) error {
	if !releaseTagRE.MatchString(tag) || tag == "." || tag == ".." {
		return errors.New("invalid release tag")
	}
	signedTag := tag
	// Horizon's nightlies use a channel-wide signature context, not their dated alias.
	if tag == "nightly" || strings.HasPrefix(tag, "nightly-") {
		signedTag = "nightly"
	}
	message := append([]byte("horizon release "+signedTag+"\n"), sums...)
	for _, line := range bytes.Split(bytes.TrimSpace(signatures), []byte{'\n'}) {
		sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(line)))
		if err == nil && ed25519.Verify(key, message, sig) {
			return nil
		}
	}
	return errors.New("release signature verification failed; refusing unsigned/untrusted download")
}

var httpClient = &http.Client{Timeout: 12 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if len(via) > 10 {
		return errors.New("too many redirects")
	}
	if req.URL.Scheme != "https" {
		return errors.New("non-HTTPS redirect denied")
	}
	return nil
}}

func get(ctx context.Context, raw string, maxBytes int64) ([]byte, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return nil, fmt.Errorf("invalid HTTPS URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "unlicensed-horizon-client/0.3 (+https://github.com/horizon-rust/horizon)")
	req.Header.Set("Accept", "application/vnd.github+json")
	r, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", r.StatusCode, u.Hostname())
	}
	if r.ContentLength > maxBytes {
		return nil, fmt.Errorf("response too large")
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("response exceeds size limit")
	}
	return b, nil
}

func findAsset(rel GHRelease, name string) (GHAsset, bool) {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return GHAsset{}, false
}
func chooseRelease(releases []GHRelease, asset string) (GHRelease, GHAsset, GHAsset, error) {
	// GitHub returns releases in reverse chronological order, not semver order.
	for _, r := range releases {
		if r.Draft || strings.HasPrefix(r.Tag, "launcher") || !releaseTagRE.MatchString(r.Tag) || r.Tag == "." || r.Tag == ".." {
			continue
		}
		archive, ok1 := findAsset(r, asset)
		sums, ok2 := findAsset(r, "SHA256SUMS")
		_, ok3 := findAsset(r, "SHA256SUMS.sig")
		if ok1 && ok2 && ok3 {
			return r, archive, sums, nil
		}
	}
	return GHRelease{}, GHAsset{}, GHAsset{}, errors.New("no Horizon GAME Windows release with ZIP and checksums in recent releases")
}
func checksumFor(sums []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.TrimPrefix(parts[1], "*") == filename {
			h := strings.ToLower(parts[0])
			if len(h) != 64 {
				return "", errors.New("invalid hash length")
			}
			if _, err := hex.DecodeString(h); err != nil {
				return "", err
			}
			return h, nil
		}
	}
	return "", fmt.Errorf("checksum not listed for %s", filename)
}
func InstallLatest(ctx context.Context, dataDir string, c *Config, progress func(string)) error {
	if runtime.GOOS != "windows" {
		return errors.New("automatic engine install currently supports Windows x64 only")
	}
	progress("Checking GitHub releases...")
	body, err := get(ctx, apiURL, 4<<20)
	if err != nil {
		return err
	}
	var releases []GHRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return err
	}
	rel, asset, sumsAsset, err := chooseRelease(releases, "horizon-windows-x64.zip")
	if err != nil {
		return err
	}
	sigAsset, exists := findAsset(rel, "SHA256SUMS.sig")
	if !exists {
		return errors.New("release has no SHA256SUMS.sig; refusing unsigned download")
	}
	for _, a := range []GHAsset{asset, sumsAsset, sigAsset} {
		parsed, e := url.Parse(a.URL)
		if e != nil || parsed.Scheme != "https" || parsed.Hostname() != "github.com" || !strings.HasPrefix(parsed.Path, "/horizon-rust/horizon/releases/download/") {
			return errors.New("unexpected download URL; aborting")
		}
	}
	progress("Selected: " + rel.Tag)
	sums, err := get(ctx, sumsAsset.URL, 64<<10)
	if err != nil {
		return err
	}
	signatures, err := get(ctx, sigAsset.URL, 16<<10)
	if err != nil {
		return err
	}
	if err := VerifyReleaseSignature(rel.Tag, sums, signatures); err != nil {
		return err
	}
	progress("Ed25519 release signature verified.")
	expected, err := checksumFor(sums, asset.Name)
	if err != nil {
		return err
	}
	progress("Downloading game binaries (no Minecraft assets)...")
	archive, err := get(ctx, asset.URL, maxArchive)
	if err != nil {
		return err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(archive))
	if hash != expected {
		return fmt.Errorf("download failed SHA-256 verification: got %s", hash)
	}
	progress("SHA-256 verified; extracting...")
	staging, err := os.MkdirTemp(dataDir, "engine-staging-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := extractSafeZIP(archive, staging); err != nil {
		return err
	}
	binary, err := findInTree(staging, ClientExecutableName())
	if err != nil {
		return err
	}
	target := filepath.Join(dataDir, "engines", rel.Tag)
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	// Releases are immutable; for reinstall, preserve old release until its replacement is staged.
	// Unique backup avoids deleting a previous interrupted installation.
	backup := ""
	if existsDir(target) {
		path, e := os.MkdirTemp(filepath.Dir(target), "engine-backup-")
		if e != nil {
			return e
		}
		_ = os.Remove(path)
		if e := os.Rename(target, path); e != nil {
			return e
		}
		backup = path
	}
	if err := os.Rename(staging, target); err != nil {
		if backup != "" {
			if e := os.Rename(backup, target); e != nil {
				return fmt.Errorf("install failed: %v; previous engine saved at %s (restore error: %v)", err, backup, e)
			}
		}
		return err
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}
	suffix, err := filepath.Rel(staging, binary)
	if err != nil {
		return err
	}
	c.ClientPath = filepath.Join(target, suffix)
	progress("Installed: " + c.ClientPath)
	return nil
}
func extractSafeZIP(archive []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return err
	}
	total := int64(0)
	if len(zr.File) > 30000 {
		return errors.New("release archive contains too many files")
	}
	for _, f := range zr.File {
		clean := filepath.Clean(filepath.FromSlash(f.Name))
		if clean == "." {
			continue
		}
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || strings.Contains(f.Name, "\\") || strings.Contains(f.Name, ":") {
			return errors.New("unsafe zip entry: " + f.Name)
		}
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return errors.New("non-regular file not allowed in releases")
		}
		total += int64(f.UncompressedSize64)
		if total > maxExtract {
			return errors.New("unpacked release too large")
		}
		path := filepath.Join(dest, clean)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			src.Close()
			return err
		}
		_, err = io.Copy(dst, io.LimitReader(src, maxExtract+1))
		src.Close()
		closeErr := dst.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
func findInTree(root, basename string) (string, error) {
	found := ""
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && strings.EqualFold(info.Name(), basename) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", errors.New("release archive has no " + basename)
	}
	return found, nil
}
