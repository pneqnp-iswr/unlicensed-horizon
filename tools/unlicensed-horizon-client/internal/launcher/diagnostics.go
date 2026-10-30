package launcher

// Probe diagnostics are intentionally conservative. The presence of a smoke
// JSON file never proves that independent data is playable in Horizon.
import (
	"encoding/json"
	"io"
	"os"
	"strings"
)

const maxProbeLogTail int64 = 128 << 10

func readProbeLogTail(path string) []byte {
	f, err := os.Open(path)
	if err != nil { return nil }
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() { return nil }
	if info.Size() > maxProbeLogTail {
		if _, err := f.Seek(-maxProbeLogTail, io.SeekEnd); err != nil { return nil }
	}
	b, _ := io.ReadAll(io.LimitReader(f, maxProbeLogTail))
	return b
}

func hasProbeSmokeJSON(raw []byte) bool {
	if len(raw) == 0 || len(raw) > 8<<20 { return false }
	var payload map[string]json.RawMessage
	return json.Unmarshal(raw, &payload) == nil && len(payload) > 0
}

func diagnoseProbe(r *ProbeResult, logTail []byte, smokeBytes []byte) {
	r.SmokeReportJSON = r.SmokeReportFound && hasProbeSmokeJSON(smokeBytes)
	log := strings.ToLower(string(logTail))
	set := func(code, action string) { r.DiagnosticCode, r.SuggestedAction = code, action }
	switch {
	case r.TimedOut:
		set("timeout", "Check process.log for the last completed startup stage; increase --timeout up to 120 only if startup is making progress.")
	case !r.ProcessStarted:
		set("start_failed", "Verify the executable exists and matches the host architecture; inspect failure and process.log.")
	case strings.Contains(log, "panicked at") || strings.Contains(log, "stack backtrace"):
		set("rust_panic", "Inspect the panic in process.log and report it with the relevant Horizon source revision.")
	case strings.Contains(log, "no such file or directory") || strings.Contains(log, "os error 2") || strings.Contains(log, "the system cannot find the file specified"):
		set("missing_file", "Use process.log to identify the exact missing path, then investigate the independent-resource generator; do not substitute proprietary game files.")
	case (strings.Contains(log, "wgpu") || strings.Contains(log, "vulkan") || strings.Contains(log, "dx12") || strings.Contains(log, "graphics adapter")) &&
		(strings.Contains(log, "failed") || strings.Contains(log, "unavailable") || strings.Contains(log, "error") || strings.Contains(log, "not found")):
		set("graphics_backend", "Check GPU driver/backend and offscreen support before treating this as a resource-format failure.")
	case r.SmokeReportFound && !r.SmokeReportJSON:
		set("invalid_smoke_json", "Inspect horizon-smoke.json; a nonempty or malformed file is not a valid smoke report.")
	case !r.ProcessExitedZero:
		set("nonzero_exit", "Inspect process.log and any Horizon smoke report to identify the failing startup stage.")
	case !r.SmokeReportFound:
		set("missing_smoke_report", "The executable exited without the requested smoke report. Check whether it supports the current smoke-test flags.")
	default:
		set("inconclusive", "A valid JSON object and zero exit are only diagnostic evidence. Verify the Rust-loader stage, rendering, and world behavior separately.")
	}
}
