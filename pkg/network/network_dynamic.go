// Copyright 2024 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package network

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"

	"github.com/nubificus/urunc/internal/constants"
	"github.com/vishvananda/netlink"
)

type DynamicNetwork struct {
}

// NetworkSetup checks if any tap device is available in the current netns. If not,
// creates a new tap device and sets tc rules between veth interface
// inside the namespace to the tap device.
// If one or more tap devices are already present in the netns, it creates a new tap device
// without tc rules and returns.
//
// FIXME: CUrrently only the first tap device can provide functional networking. The rest are "dummy" devices
// rendering the unikernels they are attached to unreachable. We need to find a proper way to handle networking
// for multiple unikernels in the same pod/network namespace
// See: https://github.com/nubificus/urunc/issues/13
func (n DynamicNetwork) NetworkSetup() (*UnikernelNetworkInfo, error) {
	err := ensureEth0Exists()
	// if eth0 does not exist in the namespace, the unikernel was spawned using ctr, so we skip the network setup
	if err != nil {
		return nil, nil
	}
	redirectLink, err := netlink.LinkByName(DefaultInterface)
	if err != nil {
		return nil, err
	}
	ifInfo, err := defaultInterfaceInfo()
	if err != nil {
		return nil, err
	}
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		return nil, err
	}
	tapIndex, err := getTapIndex()
	if err != nil {
		return nil, err
	}
	newTapName := strings.ReplaceAll(DefaultTap, "X", strconv.Itoa(tapIndex))
	newTapDevice, err := createTapDevice(newTapName, redirectLink.Attrs().MTU, uid, gid)
	if err != nil {
		return nil, err
	}
	if tapIndex == 0 {
		err = addIngressQdisc(newTapDevice)
		if err != nil {
			return nil, err
		}
		err = addIngressQdisc(redirectLink)
		if err != nil {
			return nil, err
		}
		err = addRedirectFilter(newTapDevice, redirectLink)
		if err != nil {
			return nil, err
		}
		err = addRedirectFilter(redirectLink, newTapDevice)
		if err != nil {
			return nil, err
		}
	}
	ipTemplate := fmt.Sprintf("%s/24", constants.DynamicNetworkTapIP)
	newIPAddr := strings.ReplaceAll(ipTemplate, "X", strconv.Itoa(tapIndex+1))
	ipn, err := netlink.ParseAddr(newIPAddr)
	if err != nil {
		return nil, err
	}
	err = netlink.AddrReplace(newTapDevice, ipn)
	if err != nil {
		return nil, err
	}

	err = netlink.LinkSetUp(newTapDevice)
	if err != nil {
		return nil, err
	}
	return &UnikernelNetworkInfo{
		TapDevice: newTapDevice.Attrs().Name,
		EthDevice: ifInfo,
	}, nil
}