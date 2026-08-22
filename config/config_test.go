package config

import "testing"

func TestCurrentVersion(t *testing.T) {
	t.Cleanup(func() { Version = "" })

	Version = "1.2.3"
	if got := CurrentVersion(); got != "1.2.3" {
		t.Errorf("with Version set: got %q, want %q", got, "1.2.3")
	}
	Version = ""
	if got := CurrentVersion(); got == "" {
		t.Error("without Version: got empty string")
	}
}
