/**
 * Copyright 2019 SAP SE
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package netbox

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/hosting-de-labs/go-netbox/netbox/client/virtualization"

	runtimeclient "github.com/go-openapi/runtime/client"
	netboxclient "github.com/hosting-de-labs/go-netbox/netbox/client"
	"github.com/hosting-de-labs/go-netbox/netbox/client/dcim"
	"github.com/hosting-de-labs/go-netbox/netbox/client/ipam"
	"github.com/hosting-de-labs/go-netbox/netbox/models"
)

const netboxDefaultHost = "netbox.global.cloud.sap"

type Netbox struct {
	client *netboxclient.NetBox
}

// NewDefaultHost creates a Netbox instance for the default host
func NewDefaultHost(token string) (*Netbox, error) {
	return New(netboxDefaultHost, token)
}

// New creates a Netbox instance with the host and token
func New(host, token string) (*Netbox, error) {
	client, err := client(host, token)
	if err != nil {
		return nil, err
	}
	return &Netbox{client: client}, nil
}

// Sites retrieves the all sites in the region
func (nb *Netbox) Sites(region string) ([]models.Site, error) {
	result := make([]models.Site, 0)
	params := dcim.NewDcimSitesListParams()
	params.Region = &region
	limit := int64(50)
	params.Limit = &limit

	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Dcim.DcimSitesList(params, nil)
		if err != nil {
			return nil, err
		}
		for _, site := range list.Payload.Results {
			result = append(result, *site)
		}
		if list.Payload.Next == nil {
			break
		}
	}

	return result, nil
}

// Racks retrieves all the racks with the specified role in the site
func (nb *Netbox) Racks(role string, siteID int64) ([]models.Rack, error) {
	result := make([]models.Rack, 0)
	params := dcim.NewDcimRacksListParams()
	if role != "" {
		params.Role = &role
	}
	if siteID > 0 {
		params.SiteID = &siteID
	}
	limit := int64(50)
	params.Limit = &limit

	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Dcim.DcimRacksList(params, nil)
		if err != nil {
			return nil, err
		}
		for _, rack := range list.Payload.Results {
			result = append(result, *rack)
		}
		if list.Payload.Next == nil {
			break
		}
	}

	return result, nil

}

// Servers retrieves all the servers in the rack
func (nb *Netbox) Servers(rackID int64) ([]models.DeviceWithConfigContext, error) {
	result := make([]models.DeviceWithConfigContext, 0)
	params := dcim.NewDcimDevicesListParams()
	params.RackID = &rackID
	role := "server"
	params.Role = &role
	limit := int64(50)
	params.Limit = &limit

	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Dcim.DcimDevicesList(params, nil)
		if err != nil {
			return nil, err
		}
		for _, rack := range list.Payload.Results {
			result = append(result, *rack)
		}
		if list.Payload.Next == nil {
			break
		}
	}

	return result, nil

}

//DevicesByRegion retrieves devices by region, manufacturer and status
func (nb *Netbox) DevicesByRegion(query, manufacturer, region, status string) (res []models.DeviceWithConfigContext, err error) {
	res = make([]models.DeviceWithConfigContext, 0)
	params := dcim.NewDcimDevicesListParams()
	params.WithQ(&query)
	params.WithRegion(&region)
	params.WithManufacturer(&manufacturer)
	params.WithStatus(&status)
	limit := int64(100)
	params.WithLimit(&limit)
	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Dcim.DcimDevicesList(params, nil)
		fmt.Println(*params)
		if err != nil {
			return res, err
		}
		for _, device := range list.Payload.Results {
			res = append(res, *device)
		}
		if list.Payload.Next == nil {
			break
		}
	}
	return res, err
}

//DevicesByRegion retrieves devices by region, manufacturer and status
func (nb *Netbox) DevicesByParams(params dcim.DcimDevicesListParams) (res []models.DeviceWithConfigContext, err error) {
	res = make([]models.DeviceWithConfigContext, 0)
	limit := int64(100)
	params.WithLimit(&limit)
	params.WithTimeout(30 * time.Second)

	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Dcim.DcimDevicesList(&params, nil)
		if err != nil {
			return res, err
		}
		for _, device := range list.Payload.Results {
			res = append(res, *device)
		}
		if list.Payload.Next == nil {
			break
		}
	}
	return res, err
}

//VMsByTag retrieves devices by region, manufacturer and status
func (nb *Netbox) VMsByParams(params virtualization.VirtualizationVirtualMachinesListParams) (res []models.VirtualMachineWithConfigContext, err error) {
	res = make([]models.VirtualMachineWithConfigContext, 0)
	params.WithTimeout(30 * time.Second)
	limit := int64(100)
	params.WithLimit(&limit)
	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Virtualization.VirtualizationVirtualMachinesList(&params, nil)
		if err != nil {
			return res, err
		}
		for _, vm := range list.Payload.Results {
			res = append(res, *vm)
		}
		if list.Payload.Next == nil {
			break
		}
	}
	return res, err
}

//VMsByTag retrieves devices by region, manufacturer and status
func (nb *Netbox) VMsByTag(query, status, tag string) (res []models.VirtualMachineWithConfigContext, err error) {
	res = make([]models.VirtualMachineWithConfigContext, 0)
	params := virtualization.NewVirtualizationVirtualMachinesListParams()
	params.WithQ(&query)
	params.WithStatus(&status)
	params.WithTag(&tag)
	limit := int64(100)
	params.WithLimit(&limit)
	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Virtualization.VirtualizationVirtualMachinesList(params, nil)
		if err != nil {
			return res, err
		}
		for _, vm := range list.Payload.Results {
			res = append(res, *vm)
		}
		if list.Payload.Next == nil {
			break
		}
	}
	return res, err
}

// ManagementIP retrieves the IP of the management interface for server
func (nb *Netbox) ManagementIP(serverID int64) (string, error) {

	managementInterface, err := nb.MgmtInterface(serverID, true)
	if err != nil {
		return "", err
	}

	managementIPAddress, err := nb.IPAddressByDeviceAndIntefrace(serverID, managementInterface.ID)
	if err != nil {
		return "", err
	}

	if managementIPAddress.Address == nil {
		return "", fmt.Errorf("no ip address for device %d", serverID)
	}
	ip, _, err := net.ParseCIDR(*managementIPAddress.Address)

	if err != nil {
		return "", err
	}
	return ip.String(), nil

}

// Interface retrieves the interface on the device
func (nb *Netbox) Interface(deviceID int64, interfaceName string) (*models.DeviceInterface, error) {
	params := dcim.NewDcimInterfacesListParams()
	params.DeviceID = &deviceID
	params.Name = &interfaceName

	limit := int64(1)
	params.Limit = &limit

	list, err := nb.client.Dcim.DcimInterfacesList(params, nil)
	if err != nil {
		return nil, err
	}
	if *list.Payload.Count < 1 {
		return nil, fmt.Errorf("no %s interface found for device %d", interfaceName, deviceID)
	}
	if *list.Payload.Count > 1 {
		return nil, fmt.Errorf("more than 1 %s interface found for device %d", interfaceName, deviceID)
	}

	return list.Payload.Results[0], nil

}

// MgmtInterface retrieves the management interface on the device
func (nb *Netbox) MgmtInterface(deviceID int64, mgmtOnly bool) (*models.DeviceInterface, error) {
	mgmtOnlyString := strconv.FormatBool(mgmtOnly)
	params := dcim.NewDcimInterfacesListParams()
	params.DeviceID = &deviceID
	params.MgmtOnly = &mgmtOnlyString

	limit := int64(1)
	params.Limit = &limit

	list, err := nb.client.Dcim.DcimInterfacesList(params, nil)
	if err != nil {
		return nil, err
	}
	if *list.Payload.Count < 1 {
		return nil, fmt.Errorf("no MgmtOnly=%s interface found for device %d", mgmtOnlyString, deviceID)
	}
	if *list.Payload.Count > 1 {
		return nil, fmt.Errorf("more than 1 MgmtOnly=%s interface found for device %d", mgmtOnlyString, deviceID)
	}

	return list.Payload.Results[0], nil
}

// IPAddressByDeviceAndIntefrace retrieves the IP address by device and interface
func (nb *Netbox) IPAddressByDeviceAndIntefrace(deviceID int64, interfaceID int64) (*models.IPAddress, error) {

	params := ipam.NewIPAMIPAddressesListParams()
	params.DeviceID = &deviceID
	params.InterfaceID = &interfaceID

	limit := int64(1)
	params.Limit = &limit
	list, err := nb.client.IPAM.IPAMIPAddressesList(params, nil)
	if err != nil {
		return nil, err
	}
	if *list.Payload.Count < 1 {
		return nil, fmt.Errorf(fmt.Sprintf("no ip found for device %d and interface %d", deviceID, interfaceID))
	}
	if *list.Payload.Count > 1 {
		return nil, fmt.Errorf(fmt.Sprintf("more than 1 ip found for device %d and interface %d", deviceID, interfaceID))
	}

	return list.Payload.Results[0], nil

}

// IPAddress retrieves the IPAddress by its ID
func (nb *Netbox) IPAddress(id int64) (*models.IPAddress, error) {
	params := ipam.NewIPAMIPAddressesListParams()
	ids := fmt.Sprintf("%d", id)
	params.IDIn = &ids
	limit := int64(1)
	params.Limit = &limit
	list, err := nb.client.IPAM.IPAMIPAddressesList(params, nil)
	if err != nil {
		return nil, err
	}

	if *list.Payload.Count < 1 {
		return nil, fmt.Errorf("no ip found with id %d", id)
	}
	if *list.Payload.Count > 1 {
		return nil, fmt.Errorf("more than 1 ip found for id %d", id)
	}

	return list.Payload.Results[0], nil

}

func (nb *Netbox) GetNestedDeviceIP(i *models.NestedIPAddress) (ip string, err error) {
	var ipnet net.IP
	if i == nil {
		return ip, fmt.Errorf("No IP Address found")
	}
	ipnet, _, err = net.ParseCIDR(*i.Address)
	if err != nil {
		ipnet = net.ParseIP(*i.Address)
	}

	return ipnet.String(), err
}

// RacksByRegion retrieves all the racks in the region with specified role
func (nb *Netbox) RacksByRegion(role string, region string) ([]models.Rack, error) {

	siteResults, err := nb.Sites(region)
	if err != nil {
		return nil, err
	}

	result := make([]models.Rack, 0)
	for _, s := range siteResults {
		r, err := nb.Racks(role, s.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}

	return result, nil
}

// ServersByRegion retrieves all the servers in the region with the specified rack role
func (nb *Netbox) ServersByRegion(rackRole string, region string) ([]models.DeviceWithConfigContext, error) {
	racks, err := nb.RacksByRegion(rackRole, region)
	if err != nil {
		return nil, err
	}

	results := make([]models.DeviceWithConfigContext, 0)

	for _, rack := range racks {

		r, err := nb.Servers(rack.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, r...)
	}

	return results, nil
}

// AcitveDevicesByCustomParameters retrievs all active devices with custom parameters
func (nb *Netbox) ActiveDevicesByCustomParameters(query string, params *dcim.DcimDevicesListParams) ([]models.DeviceWithConfigContext, error) {
	res := make([]models.DeviceWithConfigContext, 0)
	activeStatus := "1"
	limit := int64(100)
	params.WithStatus(&activeStatus)
	params.WithLimit(&limit)
	for {
		offset := int64(0)
		if params.Offset != nil {
			offset = *params.Offset + limit
		}
		params.Offset = &offset
		list, err := nb.client.Dcim.DcimDevicesList(params, nil)
		if err != nil {
			return res, err
		}
		for _, device := range list.Payload.Results {
			res = append(res, *device)
		}
		if list.Payload.Next == nil {
			break
		}
	}
	return res, nil
}

func client(host, token string) (*netboxclient.NetBox, error) {

	tlsClient, err := runtimeclient.TLSClient(runtimeclient.TLSClientOptions{InsecureSkipVerify: true})

	if err != nil {
		return nil, err
	}

	transport := runtimeclient.NewWithClient(host, netboxclient.DefaultBasePath, []string{"https"}, tlsClient)
	transport.DefaultAuthentication = runtimeclient.APIKeyAuth("Authorization", "header", fmt.Sprintf("Token %v", token))

	c := netboxclient.New(transport, nil)

	return c, nil

}
