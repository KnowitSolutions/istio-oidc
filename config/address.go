package config

import (
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"net"
)

func advertiseAddress(bindAddr string) (string, error) {
	host, port, err := net.SplitHostPort(bindAddr)
	if err != nil {
		err := errors.Wrap(err, "failed parsing bind address")
		return "", err
	}

	ip := net.ParseIP(host)
	if ip != nil && !ip.IsUnspecified() {
		return ip.String() + ":" + port, nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		ip = addr.(*net.IPNet).IP
		if !ip.IsLoopback() {
			return ip.String() + ":" + port, nil
		}
	}

	err = errors.New("system has no non-loopback addresses")
	return "", err
}
