package discovery

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/chyok/st/config"
)

var discoveredIPs = make(map[string]string)
var DiscoveredIPChan = make(chan [2]string)
var timestamps = make(map[string]bool)

func GetDiscoveredIPs() map[string]string {
	return discoveredIPs
}

func Send(address string, message string) error {
	timeUnixNano := time.Now().UnixNano()
	timestamp := strconv.Itoa(int(timeUnixNano))
	for i := 0; i < 3; i++ {
		addr, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			panic(err)
		}

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		_, err = conn.Write([]byte(message + "|" + timestamp))
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Millisecond * 500)
	}
	return nil

}

func Listen(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		panic(err)
	}

	var conn *net.UDPConn
	var e error

	if addr.IP.IsMulticast() {
		conn, e = net.ListenMulticastUDP("udp", nil, addr)
	} else {
		conn, e = net.ListenUDP("udp", addr)
	}
	if e != nil {
		panic(e)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			panic(err)
		}

		ip := src.IP.String()
		if config.G.LocalIP == ip {
			continue
		}
		msgs := strings.Split(string(buf[0:n]), "|")
		device, timestamp := msgs[0], msgs[1]

		if _, ok := discoveredIPs[ip]; !ok {
			fmt.Println("Find device [" + device + "] on " + ip)
			discoveredIPs[ip] = device
		}

		if _, ok := timestamps[timestamp]; !ok {
			timestamps[timestamp] = true
			if addr.IP.IsMulticast() {
				go Send(ip+":"+config.G.Port, config.G.DeviceName)
			} else {
				DiscoveredIPChan <- [2]string{device, ip}
			}
		}

	}
}
