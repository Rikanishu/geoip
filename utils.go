package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func cidrToIPv4Range(cidr string) (start uint32, end uint32, err error) {
	cidrParts := strings.Split(cidr, "/")
	if len(cidrParts) != 2 {
		return 0, 0, errors.Errorf("invalid cidr passed: %s", cidr)
	}
	start, err = ipv4toUint32(cidrParts[0])
	if err != nil {
		return 0, 0, errors.Wrapf(err, "unable to get ipv4: %s", cidrParts[0])
	}
	bits, err := strconv.ParseUint(cidrParts[1], 10, 32)
	if err != nil {
		return 0, 0, errors.Wrap(err, "unable to parse cidr %s: %v")
	}
	end = start | (0xFFFFFFFF >> bits)
	return
}

func ipv4toUint32(ipv4 string) (uint32, error) {
	var err error
	ipOctets := [4]uint64{}

	for i, v := range strings.SplitN(ipv4, ".", 4) {
		ipOctets[i], err = strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, errors.Wrapf(err, "unable to parse ip octet %v", v)
		}
	}

	result := (ipOctets[0] << 24) | (ipOctets[1] << 16) | (ipOctets[2] << 8) | ipOctets[3]

	return uint32(result), nil
}

func uint32toIPv4String(ip uint32) string {
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		(ip >> 24),
		(ip&0x00FFFFFF)>>16,
		(ip&0x0000FFFF)>>8,
		(ip & 0x000000FF),
	)

}
