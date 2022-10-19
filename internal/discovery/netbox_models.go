package discovery

import (
	ndcim "github.com/netbox-community/go-netbox/netbox/client/dcim"
	virt "github.com/netbox-community/go-netbox/netbox/client/virtualization"
)

const (
	primaryIP    int = 1
	managementIP int = 2
	loopback10   int = 3
)

type (
	customParams struct {
		CustomLabels map[string]string `yaml:"custom_labels"`
		Target       int               `yaml:"target"`
		MetricsLabel string            `yaml:"metrics_label"`
	}

	dcimDevice struct {
		ndcim.DcimDevicesListParams `yaml:",inline"`
		customParams                `yaml:",inline"`
	}

	virtualizationVM struct {
		virt.VirtualizationVirtualMachinesListParams `yaml:",inline"`
		customParams                                 `yaml:",inline"`
	}

	dcim struct {
		Devices []dcimDevice `yaml:"devices"`
	}

	virtualization struct {
		VMs []virtualizationVM `yaml:"vm"`
	}
)
