// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package net

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/StackExchange/wmi"
)

const wqlNetworkAdapter = "SELECT Description, DeviceID, Index, InterfaceIndex, MACAddress, Manufacturer, Name, NetConnectionID, ProductName, ServiceName, PhysicalAdapter, Speed FROM Win32_NetworkAdapter"

type win32NetworkAdapter struct {
	Description     *string
	DeviceID        *string
	Index           *uint32
	InterfaceIndex  *uint32
	MACAddress      *string
	Manufacturer    *string
	Name            *string
	NetConnectionID *string
	ProductName     *string
	ServiceName     *string
	PhysicalAdapter *bool
	Speed           *uint32
}

func parseNicAttrEthtool(out *bytes.Buffer) map[string][]string {
	// The out variable will now contain something that looks like the
	// following.
	//
	//Settings for eth0:
	//	Supported ports: [ TP ]
	//	Supported link modes:   10baseT/Half 10baseT/Full
	//	                        100baseT/Half 100baseT/Full
	//	                        1000baseT/Full
	//	Supported pause frame use: No
	//	Supports auto-negotiation: Yes
	//	Supported FEC modes: Not reported
	//	Advertised link modes:  10baseT/Half 10baseT/Full
	//	                        100baseT/Half 100baseT/Full
	//	                        1000baseT/Full
	//	Advertised pause frame use: No
	//	Advertised auto-negotiation: Yes
	//	Advertised FEC modes: Not reported
	//	Speed: 1000Mb/s
	//	Duplex: Full
	//	Auto-negotiation: on
	//	Port: Twisted Pair
	//	PHYAD: 1
	//	Transceiver: internal
	//	MDI-X: off (auto)
	//	Supports Wake-on: pumbg
	//	Wake-on: d
	//        Current message level: 0x00000007 (7)
	//                               drv probe link
	//	Link detected: yes

	scanner := bufio.NewScanner(out)
	// Skip the first line
	scanner.Scan()
	m := make(map[string][]string)
	var name string
	for scanner.Scan() {
		var fields []string
		if strings.Contains(scanner.Text(), ":") {
			line := strings.Split(scanner.Text(), ":")
			name = strings.TrimSpace(line[0])
			str := strings.Trim(strings.TrimSpace(line[1]), "[]")
			switch str {
			case
				"Not reported",
				"Unknown":
				continue
			}
			fields = strings.Fields(str)
		} else {
			fields = strings.Fields(strings.Trim(strings.TrimSpace(scanner.Text()), "[]"))
		}

		for _, f := range fields {
			m[name] = append(m[name], strings.TrimSpace(f))
		}
	}

	return m
}

func (i *Info) load() error {
	// Getting info from WMI
	var win32NetDescriptions []win32NetworkAdapter
	if err := wmi.Query(wqlNetworkAdapter, &win32NetDescriptions); err != nil {
		return err
	}

	i.NICs = nics(win32NetDescriptions)
	return nil
}

func nics(win32NetDescriptions []win32NetworkAdapter) []*NIC {
	// Converting into standard structures
	nics := make([]*NIC, 0)
	for _, nicDescription := range win32NetDescriptions {
		nic := &NIC{
			Name:         netDeviceName(nicDescription),
			MacAddress:   *nicDescription.MACAddress,
			MACAddress:   *nicDescription.MACAddress,
			IsVirtual:    netIsVirtual(nicDescription),
			Capabilities: []*NICCapability{},
			Speed:        netSpeed(nicDescription),
		}
		nics = append(nics, nic)
	}

	return nics
}

func netDeviceName(description win32NetworkAdapter) string {
	var name string
	if strings.TrimSpace(*description.NetConnectionID) != "" {
		name = *description.NetConnectionID
	} else {
		name = *description.Description
	}
	return name
}

func netIsVirtual(description win32NetworkAdapter) bool {
	if description.PhysicalAdapter == nil {
		return false
	}

	return !(*description.PhysicalAdapter)
}

func netSpeed(description win32NetworkAdapter) string {
	// Estimate of the current bandwidth in bits per second. For endpoints which vary in bandwidth or for those where no accurate estimation can be made, this property should contain the nominal bandwidth.
	// For more information about using uint64 values in scripts, see Scripting in WMI.

	// Need to convert to Mb/s
	if description.Speed == nil {
		return "Unknown!"
	}
	// Convert speed from bits per second to megabits per second
	speedInMbps := *description.Speed / 1_000_000
	return fmt.Sprintf("%dMb/s", speedInMbps)
}
