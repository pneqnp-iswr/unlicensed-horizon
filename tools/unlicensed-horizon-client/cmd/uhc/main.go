package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pneqnp-iswr/unlicensed-horizon/tools/unlicensed-horizon-client/internal/launcher"
)

func usage() {
	fmt.Print(`unlicensed-horizon-client 0.7  (independent local Horizon launcher)

Commands:
  doctor                     Inspect configuration and required files
  show                       Show current profile
  config [options]           Set nickname, engine and resources
  config-json BASE64         Safely import profile JSON (GUI/native arg escaping)
  install                    Install official open-source Horizon game binary
  find-resources             Locate existing prepared 26.3 game data
  audit-resources            Inspect actual loader inputs (static preflight)
  lab-init [DIR]             Generate an ORIGINAL incomplete resource experiment
  lab-audit DIR              Verify hashes, provenance and cross-file contracts
  lab-contract DIR           Inspect registry, tags, model refs and PNGs as JSON
  lab-probe --engine EXE --lab DIR  Run isolated, bounded Horizon integration probe
  legacy-worlds              Find saves left in engine folders by v0.1-v0.2
  logs                       Print game and process-log locations
  play [--wait]              Launch local session; --wait reports exit errors
  help                       Show this help

Config options:
  --name NAME                Offline nickname (3..16: letters, numbers, _)
  --client PATH              Path to horizon-client.exe
  --resources PATH           Path to prepared 26.3 game resources
  --home PATH                Game home (saves/logs/settings)
  --world NAME               Open this world (empty = title screen)
  --server HOST[:PORT]       Join a trusted offline-mode server at startup
  --distance N               Render distance 2..128
  --creative true|false      Creative for new world

This project does NOT include Minecraft assets or account entitlements.
Online-mode=true servers require an authenticated Minecraft account.
`)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	dataDir, err := launcher.DataDir()
	if err != nil {
		return err
	}
	cfg, err := launcher.Load(dataDir)
	if err != nil {
		return err
	}
	switch args[0] {
	case "help", "--help", "-h":
		usage()
		return nil
	case "show":
		fmt.Printf("name=%s\nclient=%s\nresources=%s\nhome=%s\nworld=%s\nserver=%s\ndistance=%d\ncreative=%t\nconfig=%s\n", cfg.Nickname, cfg.ClientPath, cfg.ResourcesPath, cfg.HomePath, cfg.World, cfg.Server, cfg.RenderDistance, cfg.Creative, filepath.Join(dataDir, "config.json"))
		return nil
	case "doctor":
		problems := launcher.Check(cfg)
		if len(problems) == 0 {
			fmt.Println("PRECHECK PASSED: basic binary and resource files are configured. Actual Horizon launch is not yet verified.")
			return nil
		}
		for _, x := range problems {
			fmt.Println("NEEDS ATTENTION:", x)
		}
		return fmt.Errorf("%d blocking item(s)", len(problems))
	case "find-resources":
		if len(args) != 1 {
			return fmt.Errorf("find-resources takes no options")
		}
		path, err := launcher.FindResources()
		if err != nil {
			return fmt.Errorf("no valid 26.3 prepared data found in standard locations; select folder manually")
		}
		fmt.Println(path)
		return nil
	case "legacy-worlds":
		if len(args) != 1 {
			return fmt.Errorf("legacy-worlds takes no options")
		}
		source, worlds, err := launcher.LegacyWorlds(cfg)
		if err != nil {
			return err
		}
		fmt.Println("Old engine saves:", source)
		fmt.Println("Current saves:", filepath.Join(cfg.HomePath, "saves"))
		if len(worlds) == 0 {
			fmt.Println("No recognizable legacy worlds found.")
		} else {
			fmt.Println("Found worlds:", strings.Join(worlds, ", "))
			fmt.Println("Copy each world folder into Current saves while Horizon is closed. Do not delete original backups.")
		}
		return nil
	case "audit-resources":
		if len(args) != 1 {
			return fmt.Errorf("audit-resources takes no options")
		}
		a := launcher.AuditResources(cfg.ResourcesPath)
		if err := json.NewEncoder(os.Stdout).Encode(a); err != nil {
			return err
		}
		if !a.ReadyForAttempt {
			return fmt.Errorf("resource preflight found %d issue(s)", len(a.Issues))
		}
		return nil
	case "lab-init":
		if len(args) > 2 {
			return fmt.Errorf("usage: lab-init [directory]")
		}
		dest := filepath.Join(dataDir, "open-content-lab-v0.7")
		if len(args) == 2 {
			dest = args[1]
		}
		path, err := launcher.InitOpenLab(dest)
		if err != nil {
			return err
		}
		fmt.Println("Original experimental assets generated:", path)
		fmt.Println("NOT playable. This lab is intentionally rejected by Play and doctor.")
		return nil
	case "lab-contract":
		if len(args) != 2 {
			return fmt.Errorf("usage: lab-contract DIRECTORY")
		}
		report := launcher.CheckLabContract(args[1])
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			return err
		}
		if !report.SubsetConsistent {
			return fmt.Errorf("%d structural contract issue(s)", len(report.Issues))
		}
		return nil
	case "lab-probe":
		fs := flag.NewFlagSet("lab-probe", flag.ContinueOnError)
		engine := fs.String("engine", cfg.ClientPath, "horizon-client executable")
		lab := fs.String("lab", filepath.Join(dataDir, "open-content-lab-v0.6"), "original-content laboratory")
		timeout := fs.Int("timeout", 20, "seconds (1..120)")
		frames := fs.Int("frames", 2, "test frames (1..120)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected arguments: %v", fs.Args())
		}
		report, err := launcher.ProbeOpenLab(launcher.ProbeOptions{
			Engine: *engine, Lab: *lab, Reports: filepath.Join(dataDir, "probe-reports"),
			Timeout: time.Duration(*timeout) * time.Second, Frames: *frames,
		})
		if err != nil {
			return err
		}
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Diagnostic only: actual Horizon playability remains unverified.")
		return nil
	case "lab-audit":
		if len(args) != 2 {
			return fmt.Errorf("usage: lab-audit DIRECTORY")
		}
		m, issues := launcher.AuditOpenLab(args[1])
		fmt.Printf("Lab format=%s files=%d original experimental=%t playable=%t\n", m.Format, len(m.GeneratedFiles), m.Experimental, m.ReadyForGame)
		for _, issue := range issues {
			fmt.Println("INTEGRITY ERROR:", issue)
		}
		if len(issues) > 0 {
			return fmt.Errorf("%d lab integrity problem(s)", len(issues))
		}
		fmt.Println("File checksums and subset contracts passed. Rust-loader boot and gameplay are UNVERIFIED.")
		return nil
	case "logs":
		if len(args) != 1 {
			return fmt.Errorf("logs takes no options")
		}
		fmt.Println("process=" + launcher.ProcessLog(cfg))
		fmt.Println("game=" + filepath.Join(cfg.HomePath, "logs", "latest.log"))
		return nil
	case "config-json":
		if len(args) != 2 {
			return fmt.Errorf("usage: config-json BASE64")
		}
		raw, err := base64.StdEncoding.DecodeString(args[1])
		if err != nil {
			return fmt.Errorf("profile base64: %w", err)
		}
		if len(raw) > 64<<10 {
			return fmt.Errorf("profile is too large")
		}
		var candidate launcher.Config
		if err := json.Unmarshal(raw, &candidate); err != nil {
			return fmt.Errorf("profile JSON: %w", err)
		}
		if candidate.HomePath == "" {
			return fmt.Errorf("profile home path missing")
		}
		if err := launcher.Save(dataDir, candidate); err != nil {
			return err
		}
		fmt.Println("Configuration saved.")
		return nil
	case "config":
		fs := flag.NewFlagSet("config", flag.ContinueOnError)
		name := fs.String("name", cfg.Nickname, "")
		client := fs.String("client", cfg.ClientPath, "")
		resources := fs.String("resources", cfg.ResourcesPath, "")
		home := fs.String("home", cfg.HomePath, "")
		world := fs.String("world", cfg.World, "")
		server := fs.String("server", cfg.Server, "")
		dist := fs.Int("distance", cfg.RenderDistance, "")
		creative := fs.String("creative", strconv.FormatBool(cfg.Creative), "")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unknown extra arguments: %s", strings.Join(fs.Args(), " "))
		}
		cv, err := strconv.ParseBool(*creative)
		if err != nil {
			return fmt.Errorf("invalid --creative: %w", err)
		}
		cfg.Nickname = *name
		cfg.ClientPath = *client
		cfg.ResourcesPath = *resources
		cfg.HomePath = *home
		cfg.World = *world
		cfg.Server = *server
		cfg.RenderDistance = *dist
		cfg.Creative = cv
		if cfg.HomePath == "" {
			return fmt.Errorf("--home must be a path")
		}
		if err := launcher.Save(dataDir, cfg); err != nil {
			return err
		}
		fmt.Println("Configuration saved.")
		return nil
	case "install":
		if len(args) != 1 {
			return fmt.Errorf("install takes no options")
		}
		if err := os.MkdirAll(dataDir, 0700); err != nil {
			return err
		}
		if err := launcher.InstallLatest(context.Background(), dataDir, &cfg, func(s string) { fmt.Println(s) }); err != nil {
			return err
		}
		if err := launcher.Save(dataDir, cfg); err != nil {
			return err
		}
		fmt.Println("Engine installed. Prepared Minecraft data must be supplied separately.")
		return nil
	case "play":
		wait := len(args) == 2 && args[1] == "--wait"
		if len(args) != 1 && !wait {
			return fmt.Errorf("usage: play [--wait]")
		}
		var pid int
		if wait {
			pid, err = launcher.LaunchWait(cfg)
		} else {
			pid, err = launcher.Launch(cfg)
		}
		if err != nil {
			return err
		}
		if wait {
			fmt.Printf("Horizon exited successfully (PID %d).\n", pid)
		} else {
			fmt.Printf("Started local Horizon client (PID %d) as %s\n", pid, cfg.Nickname)
		}
		return nil
	default:
		return fmt.Errorf("unknown command %q (use help)", args[0])
	}
}
