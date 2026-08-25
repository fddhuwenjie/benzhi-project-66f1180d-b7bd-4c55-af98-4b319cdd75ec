package main

import "testing"

func TestParseConfigDefaultsAndPORT(t *testing.T) {
	cfg, err := parseConfig(nil, "")
	if err != nil || cfg.addr != "127.0.0.1:19081" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	cfg, err = parseConfig(nil, "19444")
	if err != nil || cfg.addr != "127.0.0.1:19444" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	cfg, err = parseConfig([]string{"-addr=127.0.0.1:19999"}, "19444")
	if err != nil || cfg.addr != "127.0.0.1:19999" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestParseConfigRejectsUnsafeAddresses(t *testing.T) {
	for _, args := range [][]string{{"-addr=0.0.0.0:19081"}, {"-addr=127.0.0.1:0"}, {"-addr=bad"}} {
		if _, err := parseConfig(args, ""); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
	if _, err := parseConfig(nil, "not-a-port"); err == nil {
		t.Fatal("expected invalid PORT error")
	}
}
