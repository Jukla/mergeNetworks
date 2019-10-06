package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	var readNets []*net.IPNet

	if len(os.Args) != 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		fmt.Printf("Usage: %s FILE\n\n", os.Args[0])
		fmt.Println("FILE must be a sorted list of IP networks. One network each line.")
		fmt.Println("The merged list of IP networks will be printed to stdout.")
		os.Exit(0)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(fmt.Sprintf("Error opening provided config file %s.\n", os.Args[1]))
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		_, ipv4Net, err := net.ParseCIDR(line)
		if err != nil {
			continue
		}

		readNets = append(readNets, ipv4Net)
	}

	merged := true
	for merged {
		merged = false
		for i := 0; i < len(readNets)-1; i++ {
			if appendingNets(readNets[i], readNets[i+1]) {
				// Delete the following element, as the current and the following will be merged
				copy(readNets[i+1:], readNets[i+2:])
				readNets[len(readNets)-1] = nil
				readNets = readNets[:len(readNets)-1]

				readNets[i].Mask = increaseNetMask(readNets[i].Mask)
				merged = true
			}
		}
	}

	for _, rN := range readNets {
		fmt.Println(rN)
	}
}

func increaseNetMask(onm net.IPMask) net.IPMask {
	if onm[0] == 0 {
		return onm
	}

	inm := make(net.IPMask, 4)

	for i := 0; i < 4; i++ {
		if onm[i] == 255 {
			inm[i] = 255
		} else if onm[i] == 0 {
			inm[i-1] = onm[i-1] << 1
			break
		} else {
			inm[i] = onm[i] << 1
			break
		}
	}

	return inm
}

func appendingNets(netA, netB *net.IPNet) bool {
	nextBiggerMask := make(net.IPMask, 4)
	expectedNetB := make(net.IP, 4)
	extraBit := make(net.IPMask, 4)

	nextBiggerMask = increaseNetMask(netA.Mask)

	// This is the bit, which changes, when increasing the netmask:
	// 		/24	= 11111111 11111111 11111111 00000000
	// 		/23	= 11111111 11111111 11111110 00000000
	// extraBit = 00000000 00000000 00000001 00000000
	for i := 0; i < 4; i++ {
		extraBit[i] = netA.Mask[i] & ^nextBiggerMask[i]
	}

	// Get the following appending expected network:
	// 		192.168.0.0	= 11000000 10101000 00000000 00000000
	//			+
	// 		   extraBit = 00000000 00000000 00000001 00000000
	//			=
	// 		192.168.1.0	= 11000000 10101000 00000001 00000000
	for i := 0; i < 4; i++ {
		expectedNetB[i] = netA.IP[i] | extraBit[i]
	}

	for i := 0; i < 4; i++ {
		if netB.IP[i] != expectedNetB[i] {
			return false
		}
	}

	return true
}
