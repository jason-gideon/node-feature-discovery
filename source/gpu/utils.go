/*
Copyright 2020-2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gpu

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	nfdv1alpha1 "sigs.k8s.io/node-feature-discovery/pkg/apis/nfd/v1alpha1"
	"sigs.k8s.io/node-feature-discovery/pkg/ixml"
	"sigs.k8s.io/node-feature-discovery/pkg/utils/hostpath"
)

const (
	Sdkdriver       = "sdkdriver"
	DeviceCount     = "device_count"
	CstateFeature   = "cstate"
	PstateFeature   = "pstate"
	RdtFeature      = "rdt"
	SeFeature       = "se" // DEPRECATED in v0.12: will be removed in the future
	SecurityFeature = "security"
	SgxFeature      = "sgx" // DEPRECATED in v0.12: will be removed in the future
	SstFeature      = "sst"
	TopologyFeature = "topology"
)

var mandatoryDevAttrs = []string{"class", "vendor", "device", "subsystem_vendor", "subsystem_device"}
var optionalDevAttrs = []string{"sriov_totalvfs", "iommu_group/type", "iommu/intel-iommu/version"}

// Read a single PCI device attribute
// A PCI attribute in this context, maps to the corresponding sysfs file
func readSinglePciAttribute(devPath string, attrName string) (string, error) {
	data, err := os.ReadFile(filepath.Join(devPath, attrName))
	if err != nil {
		return "", fmt.Errorf("failed to read device attribute %s: %v", attrName, err)
	}
	// Strip whitespace and '0x' prefix
	attrVal := strings.TrimSpace(strings.TrimPrefix(string(data), "0x"))

	if attrName == "class" && len(attrVal) > 4 {
		// Take four first characters, so that the programming
		// interface identifier gets stripped from the raw class code
		attrVal = attrVal[0:4]
	}
	return attrVal, nil
}

// Read information of one PCI device
func readPciDevInfo(devPath string) (*nfdv1alpha1.InstanceFeature, error) {
	attrs := make(map[string]string)
	for _, attr := range mandatoryDevAttrs {
		attrVal, err := readSinglePciAttribute(devPath, attr)
		if err != nil {
			return nil, fmt.Errorf("failed to read device %s: %s", attr, err)
		}
		attrs[attr] = attrVal
	}
	for _, attr := range optionalDevAttrs {
		attrVal, err := readSinglePciAttribute(devPath, attr)
		if err == nil {
			attrs[attr] = attrVal
		}
	}
	return nfdv1alpha1.NewInstanceFeature(attrs), nil
}

// detectPci detects available PCI devices and retrieves their device attributes.
// An error is returned if reading any of the mandatory attributes fails.
func detectPci() ([]nfdv1alpha1.InstanceFeature, error) {
	sysfsBasePath := hostpath.SysfsDir.Path("bus/pci/devices")

	devices, err := os.ReadDir(sysfsBasePath)
	if err != nil {
		return nil, err
	}

	// Iterate over devices
	devInfo := make([]nfdv1alpha1.InstanceFeature, 0, len(devices))
	for _, device := range devices {
		info, err := readPciDevInfo(filepath.Join(sysfsBasePath, device.Name()))
		if err != nil {
			klog.Error(err)
			continue
		}
		devInfo = append(devInfo, *info)
	}

	return devInfo, nil
}

/*
GPU设备如下：

指定厂商的设备是否存在？

gpu.<vendor>.persent=true/false



指定厂商、指定设备类型是否存在？

gpu.<vendor>.<device型号>.persent=true/false



指定厂商，特定序号=》对应的设备类型是什么

gpu.<vendor>.<序号>=device型号（型号由厂商sdk直接获取，如果有空格将替换为k8s label能够识别的字符）
*/

// detectIluvatar detects available Iluvatar GPU devices and retrieves their device attributes.
// An error is returned if reading any of the mandatory attributes fails.
func detectIluvatar() ([]nfdv1alpha1.InstanceFeature, error) {
	//todo once init
	if err := ixml.Init(); err != nil {
		fmt.Printf("nvml error: %+v", err)
		return nil, err
	}
	defer ixml.Shutdown()

	//Get GPU device attributes by device SDK
	attrs := make(map[string]string)

	//device exist?
	devs, err := ixml.DeviceGetCount()
	if err != nil {
		return nil, err
	} else {
		attrs[DeviceCount] = strconv.FormatUint(uint64(devs), 10)
	}

	//SDK version
	attrVal, err := ixml.SystemGetDriverVersion()
	if err != nil {
		return nil, err
	} else {
		attrs[Sdkdriver] = attrVal
	}

	info := make([]nfdv1alpha1.InstanceFeature, 0)
	info = append(info, *nfdv1alpha1.NewInstanceFeature(attrs))
	return info, nil

	//Firmware version
	//todo ...

	//////
	//Read single Dev info
	//device index

	//device type

	////
	sysfsBasePath := hostpath.SysfsDir.Path("bus/pci/devices")

	devices, err := os.ReadDir(sysfsBasePath)
	if err != nil {
		return nil, err
	}

	// Iterate over devices
	devInfo := make([]nfdv1alpha1.InstanceFeature, 0, len(devices))
	for _, device := range devices {
		info, err := readPciDevInfo(filepath.Join(sysfsBasePath, device.Name()))
		if err != nil {
			klog.Error(err)
			continue
		}
		devInfo = append(devInfo, *info)
	}

	return devInfo, nil
}
