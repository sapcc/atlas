package discovery

import (
	ndcim "github.com/hosting-de-labs/go-netbox/netbox/client/dcim"
	virt "github.com/hosting-de-labs/go-netbox/netbox/client/virtualization"
)

const (
	primaryIP    int = 1
	managementIP int = 2
)

type (
	customParams struct {
		CustomLabels map[string]string `yaml:"custom_labels"`
		Target       int               `yaml:"target"`
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
