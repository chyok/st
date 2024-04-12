package config

import (
	"fmt"
	"net"
	"os"
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

func (c *Config) SetConf(port string) {
	Hostname, err := os.Hostname()
	if err != nil {
		Hostname = "unknow device"
	}
	c.DeviceName = Hostname
	c.Port = port
	c.LocalIP = getLocalIP()
	c.MulticastAddress = fmt.Sprintf("224.0.0.169:%s", port)
	c.WildcardAddress = fmt.Sprintf("0.0.0.0:%s", port)
	c.FilePath = ""
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
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
		panic("get local ip failed")
	} else if len(ips) == 1 {
		return ips[0]
	} else {
		// Select the one connected to the network
		// when there are multiple network interfaces

		// Is there a better way？
		c, err := net.Dial("udp", "8.8.8.8:80")
		if err != nil {
			return ips[0]
		}
		defer c.Close()
		return c.LocalAddr().(*net.UDPAddr).IP.String()
	}

}
