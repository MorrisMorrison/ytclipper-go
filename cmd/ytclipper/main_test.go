package main

import (
	"context"
	"log/slog"
	"testing"
)

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
