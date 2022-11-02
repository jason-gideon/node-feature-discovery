/*
Copyright 2018-2021 The Kubernetes Authors.

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
	"strings"

	"k8s.io/klog/v2"

	nfdv1alpha1 "sigs.k8s.io/node-feature-discovery/pkg/apis/nfd/v1alpha1"
	"sigs.k8s.io/node-feature-discovery/pkg/utils"
	"sigs.k8s.io/node-feature-discovery/source"
)

// Name of this feature source
const Name = "gpu"

// DeviceFeature is the name of the feature set that holds all discovered PCI devices.
// const IXDeviceFeature = "iluvatar"
const NVDeviceFeature = "nvidia"

const IXFeatureInfo = "iluvatar_info"
const IXDeviceFeature = "iluvatar_dev"

// Config holds the configuration parameters of this source.
type Config struct {
	DeviceClassWhitelist []string `json:"deviceClassWhitelist,omitempty"`
	DeviceLabelFields    []string `json:"deviceLabelFields,omitempty"`
}

// newDefaultConfig returns a new config with pre-populated defaults
func newDefaultConfig() *Config {
	return &Config{
		DeviceClassWhitelist: []string{"03", "0b40", "12"},
		DeviceLabelFields:    []string{"class", "vendor"},
	}
}

// pciSource implements the FeatureSource, LabelSource and ConfigurableSource interfaces.
type gpuSource struct {
	config   *Config
	features *nfdv1alpha1.Features
}

// Singleton source instance
var (
	src                           = gpuSource{config: newDefaultConfig()}
	_   source.FeatureSource      = &src
	_   source.LabelSource        = &src
	_   source.ConfigurableSource = &src
)

// Name returns the name of the feature source
func (s *gpuSource) Name() string { return Name }

// NewConfig method of the LabelSource interface
func (s *gpuSource) NewConfig() source.Config { return newDefaultConfig() }

// GetConfig method of the LabelSource interface
func (s *gpuSource) GetConfig() source.Config { return s.config }

// SetConfig method of the LabelSource interface
func (s *gpuSource) SetConfig(conf source.Config) {
	switch v := conf.(type) {
	case *Config:
		s.config = v
	default:
		klog.Fatalf("invalid config type: %T", conf)
	}
}

// Priority method of the LabelSource interface
func (s *gpuSource) Priority() int { return 0 }

// GetLabels method of the LabelSource interface
func (s *gpuSource) GetLabels() (source.FeatureLabels, error) {
	labels := source.FeatureLabels{}
	features := s.GetFeatures()
	vendor := "iluvatar"

	// device persent
	if len(features.Instances[IXDeviceFeature].Elements) > 0 {
		labels[vendor+".present"] = true
	}

	//sdk driver version
	if version, ok := features.Attributes[IXFeatureInfo].Elements[DriverVersion]; ok {
		labels[vendor+"."+DriverVersion] = version
	}

	// Iterate over all device classes
	for _, dev := range features.Instances[IXDeviceFeature].Elements {
		//gpu.<vendor>.<device型号>.persent=true/false	//eg: iluvatar BI-100
		attrs := dev.Attributes
		name := attrs[DeviceName]
		name = strings.TrimSpace(strings.ReplaceAll(strings.ToLower(name), vendor, ""))
		labels[vendor+"."+name+".present"] = true

		//指定厂商，特定序号=》对应的设备类型是什么
		//gpu.<vendor>.<device_inde>.<序号>=device型号
		idx := attrs[DeviceIndex]
		labels[vendor+"."+DeviceIndex+"."+idx] = name
	}
	return labels, nil
}

// Discover method of the FeatureSource interface
func (s *gpuSource) Discover() error {
	s.features = nfdv1alpha1.NewFeatures()
	devs, attrs, err := detectIluvatar()

	//devs, err := detectPci()
	if err != nil {
		return fmt.Errorf("failed to detect PCI devices: %s", err.Error())
	}
	s.features.Attributes[IXFeatureInfo] = *attrs
	s.features.Instances[IXDeviceFeature] = nfdv1alpha1.NewInstanceFeatures(devs)

	utils.KlogDump(3, "discovered pci features:", "  ", s.features)

	return nil
}

// GetFeatures method of the FeatureSource Interface
func (s *gpuSource) GetFeatures() *nfdv1alpha1.Features {
	if s.features == nil {
		s.features = nfdv1alpha1.NewFeatures()
	}
	return s.features
}

func init() {
	source.Register(&src)
}
