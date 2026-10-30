package launcher

// A controlled, opt-in integration experiment with the REAL Horizon executable.
// It deliberately has a separate command from Play: experimental resources never
// enter normal gameplay, and existing saves/configuration are never used.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type ProbeOptions struct {
	Engine  string
	Lab     string
	Reports string
	Timeout time.Duration
	Frames  int
}

type ProbeResult struct {
	Version           string `json:"version"`
	ProbeDir          string `json:"probe_dir"`
	Engine            string `json:"engine"`
	LabSource         string `json:"lab_source"`
	CopiedFiles       int    `json:"copied_files"`
	ProcessStarted    bool   `json:"process_started"`
	ProcessExitedZero bool   `json:"process_exited_zero"`
	TimedOut          bool   `json:"timed_out"`
	ElapsedMillis     int64  `json:"elapsed_millis"`
	SmokeReportFound  bool   `json:"smoke_report_found"`
	LogPath           string `json:"log_path"`
	SmokeReportPath   string `json:"smoke_report_path"`
	Failure           string `json:"failure,omitempty"`
	Conclusion        string `json:"conclusion"`
}

func safeProbeEnv(home string) []string {
	// Avoid inheriting a personal login session or any path selecting a user's
	// real game installation. The only resource root comes from --resources.
	reject := map[string]bool{
		"HORIZON_HOME": true, "HORIZON_SESSION_FILE": true,
		"HORIZON_PLAYER_NAME": true, "HORIZON_MINECRAFT_DIR": true,
		"HORIZON_LAUNCHER_HOME": true, "XDG_DATA_HOME": true,
		"XDG_CONFIG_HOME": true, "XDG_RUNTIME_DIR": true,
		"HORIZON_SERVER_THREAD": true,
		"APPDATA":               true, "LOCALAPPDATA": true,
		"USERPROFILE": true, "HOME": true,
		"TEMP": true, "TMP": true,
	}
	filtered := make([]string, 0, len(os.Environ())+5)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !reject[strings.ToUpper(key)] {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered,
		"HORIZON_HOME="+home,
		"APPDATA="+home, "LOCALAPPDATA="+home,
		"USERPROFILE="+home, "HOME="+home,
		"TEMP="+home, "TMP="+home,
		"HORIZON_PLAYER_NAME=UHC_LabTester",
		"HORIZON_NO_AUDIO=1",
		"HORIZON_LAUNCHER_NO_SELF_UPDATE=1",
		"XDG_DATA_HOME="+filepath.Join(home, "xdg-data"),
		"XDG_CONFIG_HOME="+filepath.Join(home, "xdg-config"),
		"XDG_RUNTIME_DIR="+filepath.Join(home, "xdg-runtime"),
	)
}

func saveProbeResult(path string, result ProbeResult) error {
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}

// ProbeOpenLab copies only integrity-checked, explicitly inventoried lab files
// to a new scratch tree. It NEVER writes to either the source lab or the normal
// game profile, even if the client crashes or times out.
func ProbeOpenLab(o ProbeOptions) (ProbeResult, error) {
	r := ProbeResult{Version: "0.6", Engine: o.Engine, LabSource: o.Lab,
		Conclusion: "Diagnostic probe only. Process exit and smoke-file presence are NOT proof of a playable Horizon game."}
	if o.Timeout < 1*time.Second || o.Timeout > 120*time.Second {
		return r, errors.New("timeout must be between 1 and 120 seconds")
	}
	if o.Frames < 1 || o.Frames > 120 {
		return r, errors.New("frame count must be 1..120")
	}
	if o.Engine == "" || o.Lab == "" || o.Reports == "" {
		return r, errors.New("engine, lab and reports directory are required")
	}
	absEngine, err := filepath.Abs(o.Engine)
	if err != nil {
		return r, err
	}
	stat, err := os.Lstat(absEngine)
	if err != nil || !stat.Mode().IsRegular() {
		return r, fmt.Errorf("engine is not a regular executable file: %s", absEngine)
	}
	if runtime.GOOS != "windows" && stat.Mode()&0111 == 0 {
		return r, fmt.Errorf("engine is not executable: %s", absEngine)
	}
	absLab, err := filepath.Abs(o.Lab)
	if err != nil {
		return r, err
	}
	absReports, err := filepath.Abs(o.Reports)
	if err != nil {
		return r, err
	}
	if absReports == absLab || strings.HasPrefix(absReports, absLab+string(os.PathSeparator)) || strings.HasPrefix(absLab, absReports+string(os.PathSeparator)) {
		return r, errors.New("report root must be separate from the source lab")
	}
	m, issues := AuditOpenLab(absLab)
	if len(issues) > 0 {
		return r, fmt.Errorf("lab integrity/contract failed: %s", strings.Join(issues, "; "))
	}
	if (m.Format != "uhc-open-content-lab-v2" && m.Format != "uhc-open-content-lab-v3") || !m.Experimental || m.ReadyForGame {
		return r, errors.New("only explicitly incomplete v2/v3 laboratory data may be probed")
	}
	if err := os.MkdirAll(absReports, 0700); err != nil {
		return r, err
	}
	scratch, err := os.MkdirTemp(absReports, "probe-v0.6-")
	if err != nil {
		return r, err
	}
	r.ProbeDir = scratch
	r.LogPath = filepath.Join(scratch, "process.log")
	r.SmokeReportPath = filepath.Join(scratch, "horizon-smoke.json")
	resRoot := filepath.Join(scratch, "resources")
	home := filepath.Join(scratch, "game-home")
	if err := os.MkdirAll(resRoot, 0700); err != nil {
		return r, err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return r, err
	}
	if err := os.MkdirAll(filepath.Join(home, "xdg-runtime"), 0700); err != nil {
		return r, err
	}
	paths := make([]string, 0, len(m.GeneratedFiles))
	for name := range m.GeneratedFiles {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	paths = append(paths, LabMarker)
	for _, name := range paths {
		b, e := readContainedFile(absLab, name, 32<<20)
		if e != nil {
			return r, fmt.Errorf("copy %s: %w", name, e)
		}
		dst := filepath.Join(resRoot, filepath.FromSlash(name))
		if e = os.MkdirAll(filepath.Dir(dst), 0700); e != nil {
			return r, e
		}
		if e = os.WriteFile(dst, b, 0600); e != nil {
			return r, e
		}
		r.CopiedFiles++
	}
	log, e := os.OpenFile(r.LogPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if e != nil {
		return r, e
	}
	defer log.Close()
	// A parent watchdog still kills the child if --timeout-seconds is not
	// honoured during startup. Keep it distinct from the game's own timer.
	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout+2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, absEngine,
		"--name", "UHC_LabTester", "--resources", resRoot,
		"--render-distance", "2", "--world", "UHC_LabProbe",
		"--test-terrain", "--offscreen", "--no-pause",
		"--frames", fmt.Sprint(o.Frames), "--timeout-seconds", fmt.Sprint(int(o.Timeout.Seconds())),
		"--smoke-report", r.SmokeReportPath, "--width", "640", "--height", "360")
	cmd.Dir = home
	cmd.Env = safeProbeEnv(home)
	cmd.Stdout = log
	cmd.Stderr = log
	start := time.Now()
	e = cmd.Run()
	r.ElapsedMillis = time.Since(start).Milliseconds()
	r.ProcessStarted = cmd.Process != nil
	r.TimedOut = ctx.Err() == context.DeadlineExceeded
	r.ProcessExitedZero = e == nil
	if e != nil {
		r.Failure = e.Error()
	}
	if info, statErr := os.Stat(r.SmokeReportPath); statErr == nil && info.Mode().IsRegular() && info.Size() > 0 {
		r.SmokeReportFound = true
	}
	// This result file is machine-readable and survives even a crashing client.
	if resultErr := saveProbeResult(filepath.Join(scratch, "probe-result.json"), r); resultErr != nil {
		return r, fmt.Errorf("write probe report: %w", resultErr)
	}
	// A failed game launch is an experiment outcome, not a CLI plumbing error.
	return r, nil
}
