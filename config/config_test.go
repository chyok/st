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

func TestParsePort(t *testing.T) {
	ok := map[string]string{"1": "1", "53333": "53333", "053333": "53333", "65535": "65535"}
	for in, want := range ok {
		if got, err := ParsePort(in); err != nil || got != want {
			t.Errorf("ParsePort(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "0", "-1", "65536", "80a", " 80"} {
		if got, err := ParsePort(in); err == nil {
			t.Errorf("ParsePort(%q) = %q, want error", in, got)
		}
	}
}
