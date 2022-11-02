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
	"strconv"

	"k8s.io/klog/v2"

	nfdv1alpha1 "sigs.k8s.io/node-feature-discovery/pkg/apis/nfd/v1alpha1"
	"sigs.k8s.io/node-feature-discovery/pkg/ixml"
)

const (
	DriverVersion = "driver_version"
	DeviceCount   = "device_count"
	DeviceIndex   = "device_index"
	DeviceName    = "device_name"
)

var mandatoryDevAttrs = []string{"class", "vendor", "device", "subsystem_vendor", "subsystem_device"}
var optionalDevAttrs = []string{"sriov_totalvfs", "iommu_group/type", "iommu/intel-iommu/version"}

//TODO check device vendor before Discovery

// detectIluvatar detects available Iluvatar GPU devices and retrieves their device attributes.
// An error is returned if reading any of the mandatory attributes fails.
func detectDevice() ([]nfdv1alpha1.InstanceFeature, *nfdv1alpha1.AttributeFeatureSet, error) {
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
		info, err := readSingleDeviceInfo(idx)
		if err != nil {
			klog.Error(err)
			continue
		}
		devInfo = append(devInfo, *info)
	}

	//info = append(info, *nfdv1alpha1.NewInstanceFeature(attrs))
	return devInfo, &attrFeatures, nil
}

func readSingleDeviceInfo(idx uint) (*nfdv1alpha1.InstanceFeature, error) {
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
