package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

// Config describes this st instance. main builds one and hands it to the
// parts that need it.
type Config struct {
	DeviceName string
	Port       string
	// LocalIP is the address used to talk to the LAN.
	LocalIP string
	// Addresses are the addresses to show the user, each with a QR code:
	// just the connected one when online, every candidate when offline.
	Addresses []Address
	FilePath  string
}

// ListenAddr is where the HTTP server listens: the port on all interfaces.
func (c *Config) ListenAddr() string {
	return net.JoinHostPort("0.0.0.0", c.Port)
}

// MulticastAddr is the group and port that discovery announcements use.
func (c *Config) MulticastAddr() string {
	return net.JoinHostPort("224.0.0.169", c.Port)
}

// Version is set by release builds with
// -ldflags "-X github.com/chyok/st/config.Version=<version>".
var Version string

// CurrentVersion reports Version if it was set at build time, otherwise the
// module version Go recorded (as with "go install ...@v0.2.1"), else "dev".
func CurrentVersion() string {
	if Version != "" {
		return Version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return strings.TrimPrefix(bi.Main.Version, "v")
	}
	return "dev"
}

// ParsePort checks that port is a number from 1 to 65535 and returns it in
// canonical form (so "053333" becomes "53333").
func ParsePort(port string) (string, error) {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid port %q: must be a number from 1 to 65535", port)
	}
	return strconv.Itoa(n), nil
}

// New returns the configuration for serving on port. FilePath is left empty;
// the caller sets it when sharing a file or folder.
func New(port string) (*Config, error) {
	port, err := ParsePort(port)
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown device"
	}
	addrs, err := interfaceAddresses()
	if err != nil {
		return nil, err
	}
	localIP, shown, err := chooseAddresses(addrs, routedIP())
	if err != nil {
		return nil, err
	}
	return &Config{
		DeviceName: hostname,
		Port:       port,
		LocalIP:    localIP,
		Addresses:  shown,
	}, nil
}

// Address is an IPv4 address of a network interface.
type Address struct {
	Interface string
	IP        string
}

// interfaceAddresses lists the IPv4 addresses of all up, non-loopback
// interfaces.
func interfaceAddresses() ([]Address, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var addrs []Address
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		ifAddrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range ifAddrs {
			if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				addrs = append(addrs, Address{Interface: ifi.Name, IP: ipnet.IP.String()})
			}
		}
	}
	return addrs, nil
}

// routedIP returns the local address the system would use to reach the
// internet, or "" when there is no route (offline). Dialing UDP sends no
// packets; it only asks the routing table.
func routedIP() string {
	c, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}

// chooseAddresses picks the address to use and the ones to show. With a
// route to the internet the routed address is the right one. Offline there
// is no way to tell which network the other device is on, so all are shown.
func chooseAddresses(addrs []Address, routed string) (string, []Address, error) {
	if len(addrs) == 0 {
		return "", nil, errors.New("no network address found; is this device connected to a network?")
	}
	for _, a := range addrs {
		if a.IP == routed {
			return a.IP, []Address{a}, nil
		}
	}
	return addrs[0].IP, addrs, nil
}
