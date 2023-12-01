package autodetect

import (
	"net"
	"net/netip"
)

// DetectIPv6Only detects if the pod has only IPv6 addresses assigned and no IPv4 addresses.
func DetectIPv6Only(interfaces []string) bool {
	foundIPv4 := false
	foundIPv6 := false

	for _, name := range interfaces {
		inf, err := net.InterfaceByName(name)
		if err != nil {
			continue
		}

		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}

		for _, a := range addrs {
			prefix, err := netip.ParsePrefix(a.String())
			if err != nil {
				continue
			}

			if prefix.Addr().Is4() {
				foundIPv4 = true
			} else if prefix.Addr().Is6() {
				foundIPv6 = true
			}
		}
	}

	return foundIPv6 && !foundIPv4
}
