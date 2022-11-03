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
	DriverVersion   = "driver_version"
	DeviceCount     = "device_count"
	DeviceIndex     = "device_index"
	DeviceName      = "device_name"
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
func detectIluvatar() ([]nfdv1alpha1.InstanceFeature, *nfdv1alpha1.AttributeFeatureSet, error) {
	//todo once init ...
	if err := ixml.Init(); err != nil {
		fmt.Printf("nvml error: %+v", err)
		return nil, nil, err
	}
	defer ixml.Shutdown()

	//Get GPU device attributes by device SDK
	//Get Attribute

	//SDK version
	attrs := make(map[string]string)
	driverVersion, err := ixml.SystemGetDriverVersion()
	if err != nil {
		return nil, nil, err
	} else {
		attrs[DriverVersion] = driverVersion
	}

	//Firmware version
	//todo ...

	//device exist?
	devs, err := ixml.DeviceGetCount()
	if err != nil {
		return nil, nil, err
	} else {
		attrs[DeviceCount] = strconv.FormatUint(uint64(devs), 10)
	}

	attrFeatures := nfdv1alpha1.NewAttributeFeatures(attrs)

	//////
	//Get Instance
	devInfo := make([]nfdv1alpha1.InstanceFeature, 0)
	//Read single Dev info
	for idx := uint(0); idx < devs; idx++ {
		info, err := readSingleIluvaterDeviceInfo(idx)
		if err != nil {
			klog.Error(err)
			continue
		}
		devInfo = append(devInfo, *info)
	}

	//info = append(info, *nfdv1alpha1.NewInstanceFeature(attrs))
	return devInfo, &attrFeatures, nil
}

func readSingleIluvaterDeviceInfo(idx uint) (*nfdv1alpha1.InstanceFeature, error) {
	attrs := make(map[string]string)
	//device index
	dev, err := ixml.DeviceGetHandleByIndex(idx)
	if err != nil {
		return nil, fmt.Errorf("failed to read iluvater device idx %d: %s", idx, err)
	}
	attrs[DeviceIndex] = strconv.FormatUint(uint64(idx), 10)

	//device name
	name, err := dev.DeviceGetName()
	if err != nil {
		return nil, fmt.Errorf("failed to read iluvater device idx %d: %s", idx, err)
	}
	attrs[DeviceName] = name

	//todo anothers..

	return nfdv1alpha1.NewInstanceFeature(attrs), nil
}
