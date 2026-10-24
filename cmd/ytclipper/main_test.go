package main

import "testing"

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
