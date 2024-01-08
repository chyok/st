package util

import (
	"net"
)

func In[T comparable](v T, arr []T) bool {
	for _, data := range arr {
		if v == data {
			return true
		}
	}
	return false
}

func GetLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var ips []string
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip.To4() != nil {
			ips = append(ips, ip.To4().String())
		}

	}
	return ips, nil
}
