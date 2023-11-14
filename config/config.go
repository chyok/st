package config

import (
	"net"
	"os"
)

type Config struct {
	DeviceName       string
	Port             string
	LocalIP          string
	MulticastAddress string
	WildcardAddress  string
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
	c.MulticastAddress = "224.0.0.1" + ":" + port
	c.WildcardAddress = "0.0.0.0" + ":" + port
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	panic("get local ip failed")
}
