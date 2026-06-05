package netutil

import (
	"net"

	"github.com/pkg/errors"
)

func OutboundIP() (net.IP, error) {
	if ip, err := outboundIPByUDP(); err == nil {
		return ip, nil
	}
	return outboundIPByInterface()
}

func outboundIPByUDP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	addr := conn.LocalAddr().(*net.UDPAddr)
	if addr.IP.IsGlobalUnicast() {
		return addr.IP, nil
	}
	return nil, errors.New("netutil: udp dial returned non-unicast IP")
}

func outboundIPByInterface() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "netutil: list interfaces")
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || !ip.IsGlobalUnicast() {
				continue
			}
			if ip.To4() != nil {
				return ip, nil
			}
		}
	}
	return nil, errors.New("netutil: no valid outbound IP found")
}
