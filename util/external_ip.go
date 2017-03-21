//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"errors"
	"net"
)

// helper function to determine the external IP address of a query node -
// used to create a name for the query node in NewQueryNode function.
func ExternalIP() (string, error) {
	nics, errs := ExternalNICs()
	if len(nics) == 0 && len(errs) == 0 {
		return "", errors.New("Not connected to the network")
	}

	// we return the first error
	if len(errs) > 0 {
		return "", errs[0]
	}
	ip := nics[0].IP()
	return ip, nil
}

// helper object to determine IP and state of the local network interfaces
// where we do network name intensive operations (clustering for instance)
// it's a better option to cache the name and redetermine the local IP
// only if the state of the chosen NIC has changed.
type NetworkInterface struct {
	hardware *net.Interface
	addrs    int
	ip       string
}

func ExternalNICs() ([]*NetworkInterface, []error) {
	var networkInterfaces []*NetworkInterface
	var errs []error

	ifaces, err := net.Interfaces()
	if err != nil {
		return networkInterfaces, errs
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()

		// no addresses?
		if err != nil {

			errs = append(errs, err)
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
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			// we want the pointer to the actual interface, if we want to be
			// able to test flags, not a pointer to our for variable...
			hardware, err := net.InterfaceByName(iface.Name)
			if err == nil {
				networkInterface := &NetworkInterface{
					hardware: hardware,
					ip:       ip.String(),
					addrs:    len(addrs),
				}
				networkInterfaces = append(networkInterfaces, networkInterface)
				break
			} else {
				errs = append(errs, err)
			}
		}
	}
	return networkInterfaces, errs
}

func (this *NetworkInterface) IP() string {
	return this.ip
}

func (this *NetworkInterface) Up() bool {
	if this.hardware == nil {
		return false
	}

	// FlagUp is not a good enough indicator that the interface might be down:
	// ifconfig (on OSX Yosemite at least) reports the UP flag for wifi adapters
	// which are powered off.
	// We compoud the flags test with checking the number of addresses.
	// It's not as efficient as we were hoping, but not as bad as scanning all
	// interfaces and all addresses over and over again.
	addrs, _ := this.hardware.Addrs()
	return this.hardware.Flags&net.FlagUp != 0 && len(addrs) == this.addrs
}
