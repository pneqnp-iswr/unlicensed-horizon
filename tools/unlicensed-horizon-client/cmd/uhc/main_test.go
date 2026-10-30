package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pneqnp-iswr/unlicensed-horizon/tools/unlicensed-horizon-client/internal/launcher"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONProfileEmptyArguments(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("UHC_DATA_DIR", dir)
	c := launcher.DefaultConfig(dir)
	c.Nickname = "Tester_1"
	c.World = ""
	c.Server = ""
	c.ClientPath = "C:\\Path With Spaces\\horizon-client.exe"
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.StdEncoding.EncodeToString(b)
	if err := run([]string{"config-json", payload}); err != nil {
		t.Fatal(err)
	}
	loaded, err := launcher.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.World != "" || loaded.Server != "" || loaded.ClientPath != c.ClientPath {
		t.Fatalf("round trip broken: %+v", loaded)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"config-json", "not base64"}); err == nil {
		t.Fatal("accepted invalid profile")
	}
}

func TestCLIOriginalResourceLab(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("UHC_DATA_DIR", dir)
	lab := filepath.Join(dir, "open-content-lab-v0.6")
	if err := run([]string{"lab-init"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"lab-audit", lab}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"lab-init"}); err == nil {
		t.Fatal("overwrote existing lab")
	}
	if err := run([]string{"config", "--resources", lab}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"doctor"}); err == nil {
		t.Fatal("doctor accepted experimental unplayable assets")
	}
	if err := run([]string{"play", "--wait"}); err == nil {
		t.Fatal("play accepted experimental unplayable assets")
	}
}
