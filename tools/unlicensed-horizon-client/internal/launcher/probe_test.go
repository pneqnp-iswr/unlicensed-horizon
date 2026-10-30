package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSafeProbeEnvRedactsAuthenticationAndHome(t *testing.T) {
	t.Setenv("HORIZON_SESSION_FILE", "secret-token.json")
	t.Setenv("HORIZON_HOME", "/real/home")
	t.Setenv("APPDATA", "/real/appdata")
	tmp := t.TempDir()
	env := safeProbeEnv(tmp)
	found := map[string]string{}
	for _, entry := range env {
		key, val, _ := strings.Cut(entry, "=")
		found[strings.ToUpper(key)] = val
		if strings.Contains(entry, "secret-token") || strings.Contains(entry, "/real/home") || strings.Contains(entry, "/real/appdata") {
			t.Errorf("unsafe environment leaked: %q", entry)
		}
	}
	if _, ok := found["HORIZON_SESSION_FILE"]; ok {
		t.Fatal("session variable inherited")
	}
	for _, key := range []string{"HORIZON_HOME", "APPDATA", "LOCALAPPDATA", "USERPROFILE", "HOME", "TEMP", "TMP"} {
		if found[key] != tmp {
			t.Errorf("%s=%q, expected scratch", key, found[key])
		}
	}
}

func makeTestProbe(t *testing.T, script string) (ProbeOptions, string) {
	t.Helper()
	root := t.TempDir()
	lab := filepath.Join(root, "lab")
	if _, err := InitOpenLab(lab); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "stub.sh")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"+script+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return ProbeOptions{Engine: exe, Lab: lab, Reports: filepath.Join(root, "reports"), Timeout: 2 * time.Second, Frames: 2}, root
}

func TestProbeLabIsolationAndReports(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture, Windows compilation checked separately")
	}
	o, _ := makeTestProbe(t, `
      printf '%s\n' "$PWD" > invocation.txt
      printf '%s\n' "$HORIZON_HOME" >> invocation.txt
      printf '%s\n' "$APPDATA" >> invocation.txt
      for p in "$@"; do
        if [ "$seen" = "yes" ]; then printf '{"mock":true}' > "$p"; break; fi
        if [ "$p" = "--smoke-report" ]; then seen=yes; fi
      done
      echo "mock process result" >&2
      exit 0`)
	r, err := ProbeOpenLab(o)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ProcessStarted || !r.ProcessExitedZero || !r.SmokeReportFound || r.TimedOut {
		t.Fatalf("unexpected result: %+v", r)
	}
	if r.CopiedFiles != 83 {
		t.Fatalf("expected 82 files + marker, got %d", r.CopiedFiles)
	}
	if _, err := os.Stat(filepath.Join(r.ProbeDir, "game-home", "invocation.txt")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(r.ProbeDir, "game-home", "invocation.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if !strings.HasPrefix(line, r.ProbeDir) {
			t.Fatalf("process escaped scratch: %s", line)
		}
	}
	saved, err := os.ReadFile(filepath.Join(r.ProbeDir, "probe-result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded ProbeResult
	if err = json.Unmarshal(saved, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ProbeDir != r.ProbeDir {
		t.Fatal("invalid saved probe report")
	}
	if !strings.Contains(decoded.Conclusion, "NOT proof") {
		t.Fatal("report overstated conclusions")
	}
}

func TestProbeLabFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	o, _ := makeTestProbe(t, "exit 7")
	r, err := ProbeOpenLab(o)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ProcessStarted || r.ProcessExitedZero || r.Failure == "" {
		t.Fatalf("expected recorded failed process: %+v", r)
	}
	if err = os.WriteFile(filepath.Join(o.Lab, "facts.json"), []byte(`{"bad":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = ProbeOpenLab(o); err == nil {
		t.Fatal("ran damaged lab")
	}
	o.Timeout = 0
	if _, err = ProbeOpenLab(o); err == nil {
		t.Fatal("accepted zero timeout")
	}
}

func TestProbeLabCannotWriteInsideSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	o, _ := makeTestProbe(t, "exit 0")
	o.Reports = filepath.Join(o.Lab, "probe-reports")
	if _, err := ProbeOpenLab(o); err == nil {
		t.Fatal("accepted report root inside source lab")
	}
}

func TestV3EnvironmentContractRejectsCorruptTimeline(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lab")
	if _, err := InitOpenLab(root); err != nil {
		t.Fatal(err)
	}
	check := CheckLabContract(root)
	if !check.SubsetConsistent {
		t.Fatalf("generated environment is invalid: %v", check.Issues)
	}
	timeline := filepath.Join(root, "data", "minecraft", "timeline", "day.json")
	if err := os.WriteFile(timeline, []byte(`{"period_ticks":0,"tracks":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if CheckLabContract(root).SubsetConsistent {
		t.Fatal("accepted invalid environment timeline")
	}
	if _, issues := AuditOpenLab(root); len(issues) == 0 {
		t.Fatal("integrity audit missed corrupted timeline")
	}
}
