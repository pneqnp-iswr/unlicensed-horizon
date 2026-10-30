package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Arguments(c Config) []string {
	out := []string{"--name", c.Nickname, "--resources", c.ResourcesPath, "--render-distance", fmt.Sprint(c.RenderDistance)}
	if c.Server != "" {
		out = append(out, "--server", c.Server)
	}
	if c.World != "" {
		out = append(out, "--world", c.World)
	}
	if c.Creative {
		out = append(out, "--creative")
	}
	return out
}

// ProcessLog returns the file receiving Horizon's stdout and stderr. Horizon
// separately writes its structured game log to <home>/logs/latest.log.
func ProcessLog(c Config) string { return filepath.Join(c.HomePath, "logs", "launcher-process.log") }

func commandWithLog(c Config) (*exec.Cmd, *os.File, error) {
	if issues := Check(c); len(issues) > 0 {
		return nil, nil, fmt.Errorf("cannot launch: %v", issues)
	}
	// Horizon reads saves/ and options.txt from the PROCESS working directory.
	// Running from the engine directory would put user worlds inside an updatable release.
	if err := os.MkdirAll(filepath.Join(c.HomePath, "logs"), 0700); err != nil {
		return nil, nil, err
	}
	exe, err := filepath.Abs(c.ClientPath)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(ProcessLog(c), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, nil, err
	}
	home, err := filepath.Abs(c.HomePath)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	res, err := filepath.Abs(c.ResourcesPath)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	launching := c
	launching.ResourcesPath = res
	cmd := exec.Command(exe, Arguments(launching)...)
	cmd.Dir = home
	cmd.Env = append(os.Environ(), "HORIZON_HOME="+home)
	cmd.Stdout = f
	cmd.Stderr = f
	return cmd, f, nil
}

// Launch starts an independent game process and returns as soon as it is spawned.
func Launch(c Config) (int, error) {
	cmd, f, err := commandWithLog(c)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return pid, err
	}
	return pid, nil
}

// LaunchWait leaves the CLI process attached until Horizon terminates and
// reports startup errors and nonzero game exit codes to the parent launcher.
func LaunchWait(c Config) (int, error) {
	cmd, f, err := commandWithLog(c)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		return pid, fmt.Errorf("Horizon exited unsuccessfully (see %s): %w", ProcessLog(c), err)
	}
	return pid, nil
}
