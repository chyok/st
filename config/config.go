package config

import (
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strings"
)

type Config struct {
	DeviceName       string
	Port             string
	LocalIP          string
	MulticastAddress string
	WildcardAddress  string
	FilePath         string
}

var G Config

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

func (c *Config) SetConf(port string) error {
	Hostname, err := os.Hostname()
	if err != nil {
		Hostname = "unknown device"
	}
	c.DeviceName = Hostname
	c.Port = port
	c.LocalIP, err = getLocalIP()
	if err != nil {
		return err
	}
	c.MulticastAddress = fmt.Sprintf("224.0.0.169:%s", port)
	c.WildcardAddress = fmt.Sprintf("0.0.0.0:%s", port)
	c.FilePath = ""
	return nil
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	var ips []string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("get local ip failed")
	} else if len(ips) == 1 {
		return ips[0], nil
	} else {
		// Select the one connected to the network
		// when there are multiple network interfaces

		// Is there a better way？
		c, err := net.Dial("udp", "8.8.8.8:80")
		if err != nil {
			return ips[0], nil
		}
		defer c.Close()
		return c.LocalAddr().(*net.UDPAddr).IP.String(), nil
	}

}
