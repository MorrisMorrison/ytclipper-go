package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRedactProxy(t *testing.T) {
	tests := map[string]string{
		"http://user:secret@proxy:8118": "http://%2A%2A%2A@proxy:8118",
		"http://proxy:8118":             "http://proxy:8118",
		"socks5h://u:p@10.0.0.1:1080":   "socks5h://%2A%2A%2A@10.0.0.1:1080",
		"::not a url":                   "::not a url",
	}
	for in, want := range tests {
		if got := redactProxy(in); got != want {
			t.Errorf("redactProxy(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestCheckCommand(t *testing.T) {
	dir := t.TempDir()
	name := "ytclipper-fake-tool"
	file := name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	if err := checkCommand(name); err != nil {
		t.Errorf("checkCommand(%q) = %v; want nil for a command on PATH", name, err)
	}
	if err := checkCommand("ytclipper-missing-tool"); err == nil {
		t.Error("checkCommand must fail for a command that isn't on PATH")
	}
}

func TestIsSocksProxy(t *testing.T) {
	tests := map[string]bool{
		"socks5h://10.0.0.1:1080": true,
		"socks5://host:1080":      true,
		"SOCKS4://host:1080":      true,
		"http://10.0.0.1:8118":    false,
		"https://proxy:443":       false,
		"":                        false,
	}
	for proxy, want := range tests {
		if got := isSocksProxy(proxy); got != want {
			t.Errorf("isSocksProxy(%q) = %v; want %v", proxy, got, want)
		}
	}
}

func TestNewLogger(t *testing.T) {
	ctx := context.Background()
	if newLogger("text", false).Enabled(ctx, slog.LevelDebug) {
		t.Error("debug must be off unless YTCLIPPER_DEBUG is set")
	}
	if !newLogger("JSON", true).Enabled(ctx, slog.LevelDebug) {
		t.Error("debug must be on with YTCLIPPER_DEBUG")
	}
	if _, ok := newLogger("json", false).Handler().(*slog.JSONHandler); !ok {
		t.Error(`format "json" must use the JSON handler`)
	}
	if _, ok := newLogger("", false).Handler().(*slog.TextHandler); !ok {
		t.Error("the default format must be text")
	}
}
