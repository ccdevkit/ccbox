package constants

import "testing"

func TestHookRequestType(t *testing.T) {
	if HookRequestType != "hook" {
		t.Errorf("HookRequestType = %q, want %q", HookRequestType, "hook")
	}
}

func TestHookProxyBinaryName(t *testing.T) {
	if HookProxyBinaryName != "cchookproxy" {
		t.Errorf("HookProxyBinaryName = %q, want %q", HookProxyBinaryName, "cchookproxy")
	}
}

func TestHookDialTimeoutSec(t *testing.T) {
	if HookDialTimeoutSec != 2 {
		t.Errorf("HookDialTimeoutSec = %d, want %d", HookDialTimeoutSec, 2)
	}
}

func TestHookResponseTimeoutSec(t *testing.T) {
	if HookResponseTimeoutSec != 10 {
		t.Errorf("HookResponseTimeoutSec = %d, want %d", HookResponseTimeoutSec, 10)
	}
}

func TestExistingConstants(t *testing.T) {
	if ExecRequestType != "exec" {
		t.Errorf("ExecRequestType = %q, want %q", ExecRequestType, "exec")
	}
	if LogRequestType != "log" {
		t.Errorf("LogRequestType = %q, want %q", LogRequestType, "log")
	}
}
