package launcher

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestSignedNightlyAssetsNeverNeedGitHubAPI(t *testing.T) {
	rel, archive, sums, signature := signedNightlyAssets()
	if rel.Tag != "nightly" || !rel.Prerelease {
		t.Fatalf("unexpected release: %+v", rel)
	}
	expected := map[string]string{
		archive.Name: "horizon-windows-x64.zip",
		sums.Name: "SHA256SUMS",
		signature.Name: "SHA256SUMS.sig",
	}
	if len(expected) != 3 {
		t.Fatalf("missing or duplicated required signed release assets: %+v", expected)
	}
	for _, a := range []GHAsset{archive, sums, signature} {
		u, err := url.Parse(a.URL)
		if err != nil || u.Scheme != "https" || u.Host != "github.com" {
			t.Fatalf("unsafe release asset URL %q: %v", a.URL, err)
		}
		wantPath := "/horizon-rust/horizon/releases/download/nightly/" + a.Name
		if u.Path != wantPath || strings.Contains(a.URL, "api.github.com") {
			t.Fatalf("unexpected release path %q, expected %q", u.Path, wantPath)
		}
		if _, ok := findAsset(rel, a.Name); !ok {
			t.Errorf("missing named asset %s", a.Name)
		}
	}
}

func TestNightlySignatureChecksAreNotBypassed(t *testing.T) {
	seed := []byte("0123456789abcdef0123456789abcdef")
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	sums := []byte(strings.Repeat("a", 64) + "  horizon-windows-x64.zip\n")
	message := append([]byte("horizon release nightly\n"), sums...)
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(private, message)))
	if err := verifyReleaseWithKey("nightly", sums, sig, public); err != nil {
		t.Fatal(err)
	}
	if err := verifyReleaseWithKey("nightly-2026-11-03", sums, sig, public); err != nil {
		t.Fatal("nightly aliases should use the signed nightly context: ", err)
	}
	tampered := append([]byte(nil), sums...)
	tampered[0] = 'b'
	if err := verifyReleaseWithKey("nightly", tampered, sig, public); err == nil {
		t.Fatal("tampered checksum manifest accepted")
	}
	if err := verifyReleaseWithKey("other-release", sums, sig, public); err == nil {
		t.Fatal("signature accepted under unrelated release tag")
	}
	hash, err := checksumFor(sums, "horizon-windows-x64.zip")
	if err != nil || hash != strings.Repeat("a", 64) {
		t.Fatalf("missing or invalid digest: %s %v", hash, err)
	}
}
