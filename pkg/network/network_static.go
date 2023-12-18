// Copyright 2023 Nubificus LTD.

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
	"os/user"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
)

const StaticIPAddr = "10.10.1.1"

type StaticNetwork struct {
	// ftaxnv tap device me ip 10.10.1.1
	// sto cli tou unikernel dinw 10.10.10.2 + inject to idio sto queue-proxy container san env
}

func (n StaticNetwork) NetworkSetup() (*UnikernelNetworkInfo, error) {
	err := ensureEth0Exists()
	if err != nil {
		return nil, err
	}
	redirectLink, err := netlink.LinkByName(DefaultInterface)
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
	newTapName := strings.ReplaceAll(DefaultTap, "X", "0")
	newTapDevice, err := createTapDevice(newTapName, redirectLink.Attrs().MTU, uid, gid)
	if err != nil {
		return nil, err
	}
	ipn, err := netlink.ParseAddr(StaticIPAddr)
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
		EthDevice: Interface{
			IP:             "10.10.10.2",
			DefaultGateway: "10.10.10.1",
			Mask:           "255.255.255.0",
			Interface:      "eth0", // or tap0_urunc?
		},
	}, nil
}
