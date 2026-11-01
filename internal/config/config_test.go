package config

import (
	"slices"
	"testing"
)

// Not parallel: it sets the package-level Version.
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
	t.Parallel()
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

func TestChooseAddresses(t *testing.T) {
	t.Parallel()
	wifi := Address{Interface: "en0", IP: "192.168.1.5"}
	cable := Address{Interface: "en7", IP: "10.0.0.2"}

	cases := []struct {
		name      string
		addrs     []Address
		routed    string
		wantLocal string
		wantShown []Address
	}{
		{"online picks the routed address", []Address{cable, wifi}, "192.168.1.5", "192.168.1.5", []Address{wifi}},
		{"offline shows every address", []Address{wifi, cable}, "", "192.168.1.5", []Address{wifi, cable}},
		{"single address offline", []Address{cable}, "", "10.0.0.2", []Address{cable}},
		{"route via an unlisted address", []Address{wifi, cable}, "172.16.0.9", "192.168.1.5", []Address{wifi, cable}},
	}
	for _, c := range cases {
		local, shown, err := chooseAddresses(c.addrs, c.routed)
		if err != nil || local != c.wantLocal || !slices.Equal(shown, c.wantShown) {
			t.Errorf("%s: got %q, %v, %v; want %q, %v", c.name, local, shown, err, c.wantLocal, c.wantShown)
		}
	}

	if _, _, err := chooseAddresses(nil, ""); err == nil {
		t.Error("no addresses: want error")
	}
}
