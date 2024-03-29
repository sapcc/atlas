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
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/netbox-community/go-netbox/netbox/client/virtualization"

	runtimeclient "github.com/go-openapi/runtime/client"
	netboxclient "github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/dcim"
	"github.com/netbox-community/go-netbox/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/netbox/models"
)

const netboxDefaultHost = "netbox.global.cloud.sap"

type Netbox struct {
	client *netboxclient.NetBoxAPI
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
	params.WithContext(context.Background())
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
func (nb *Netbox) Racks(role string, siteID string) ([]models.Rack, error) {
	result := make([]models.Rack, 0)
	params := dcim.NewDcimRacksListParams()
	params.WithContext(context.Background())
	if role != "" {
		params.Role = &role
	}
	if siteID != "" {
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
func (nb *Netbox) Servers(rackID string) ([]models.DeviceWithConfigContext, error) {
	result := make([]models.DeviceWithConfigContext, 0)
	params := dcim.NewDcimDevicesListParams()
	params.WithContext(context.Background())
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
	params.WithContext(context.Background())
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
	params.WithContext(context.Background())

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

//DevicesByRegion retrieves devices by region, manufacturer and status
func (nb *Netbox) DeviceByParams(params dcim.DcimDevicesListParams) (res models.DeviceWithConfigContext, err error) {
	limit := int64(1)
	params.WithLimit(&limit)
	params.WithContext(context.Background())
	params.WithTimeout(30 * time.Second)

	list, err := nb.client.Dcim.DcimDevicesList(&params, nil)
	if err != nil {
		return res, err
	}
	for _, device := range list.Payload.Results {
		res = *device
	}

	return res, err
}

//VMsByTag retrieves devices by region, manufacturer and status
func (nb *Netbox) VMsByParams(params virtualization.VirtualizationVirtualMachinesListParams) (res []models.VirtualMachineWithConfigContext, err error) {
	res = make([]models.VirtualMachineWithConfigContext, 0)
	params.WithTimeout(30 * time.Second)
	limit := int64(100)
	params.WithLimit(&limit)
	params.WithContext(context.Background())
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
	params.WithContext(context.Background())
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
func (nb *Netbox) ManagementIPs(serverID string) (ips []string, err error) {

	managementInterface, err := nb.MgmtInterface(serverID, true)
	if err != nil {
		return
	}
	ips = make([]string, 0)
	for _, intf := range managementInterface {
		managementIPAddress, err := nb.IPAddressByDeviceAndIntefrace(serverID, strconv.FormatInt(intf.ID, 10))
		if err != nil {
			return ips, err
		}
		if managementIPAddress.Address == nil {
			return ips, fmt.Errorf("no ip address for device %s", serverID)
		}
		ip, _, err := net.ParseCIDR(*managementIPAddress.Address)

		if err != nil {
			return ips, err
		}
		ips = append(ips, ip.String())
	}
	return
}

// DeviceInterfaceNameIPs retrieves the IP of the named interface for server
func (nb *Netbox) DeviceInterfaceNameIPs(name, deviceID string) (ips []string, err error) {
	var intf *models.Interface
	var ipAddr *models.IPAddress
	intf, err = nb.Interface(deviceID, name)
	if err != nil {
		return ips, fmt.Errorf("Error getting interface from device: %s. Error: %s", deviceID, err.Error())
	}
	ipAddr, err = nb.IPAddressByDeviceAndIntefrace(deviceID, strconv.FormatInt(intf.ID, 10))
	if err != nil {
		return ips, fmt.Errorf("Error getting ip from device interface: %s. Error: %s", deviceID, err.Error())
	}
	var ip net.IP
	ip, _, err = net.ParseCIDR(*ipAddr.Address)
	if err != nil {
		return ips, fmt.Errorf("Error parsing ip from device: %s. Error: %s", deviceID, err.Error())
	}
	ips = append(ips, ip.String())

	return
}

// Interface retrieves the interface on the device
func (nb *Netbox) Interface(deviceID string, interfaceName string) (*models.Interface, error) {
	params := dcim.NewDcimInterfacesListParams()
	params.DeviceID = &deviceID
	params.Name = &interfaceName
	params.WithContext(context.Background())

	limit := int64(1)
	params.Limit = &limit

	list, err := nb.client.Dcim.DcimInterfacesList(params, nil)
	if err != nil {
		return nil, err
	}
	if *list.Payload.Count < 1 {
		return nil, fmt.Errorf("no %s interface found for device %s", interfaceName, deviceID)
	}
	if *list.Payload.Count > 1 {
		return nil, fmt.Errorf("more than 1 %s interface found for device %s", interfaceName, deviceID)
	}

	return list.Payload.Results[0], nil

}

// MgmtInterface retrieves the management interface on the device
func (nb *Netbox) MgmtInterface(deviceID string, mgmtOnly bool) ([]*models.Interface, error) {
	mgmtOnlyString := strconv.FormatBool(mgmtOnly)
	params := dcim.NewDcimInterfacesListParams()
	params.DeviceID = &deviceID
	params.MgmtOnly = &mgmtOnlyString
	params.WithContext(context.Background())

	limit := int64(2)
	params.Limit = &limit

	list, err := nb.client.Dcim.DcimInterfacesList(params, nil)
	if err != nil {
		return nil, err
	}

	return list.Payload.Results, nil
}

// IPAddressByDeviceAndIntefrace retrieves the IP address by device and interface
func (nb *Netbox) IPAddressByDeviceAndIntefrace(deviceID string, interfaceID string) (*models.IPAddress, error) {

	params := ipam.NewIpamIPAddressesListParams()
	params.DeviceID = &deviceID
	params.InterfaceID = &interfaceID
	params.WithContext(context.Background())

	limit := int64(1)
	params.Limit = &limit
	list, err := nb.client.Ipam.IpamIPAddressesList(params, nil)
	if err != nil {
		return nil, err
	}

	if *list.Payload.Count < 1 {
		return nil, fmt.Errorf(fmt.Sprintf("no ip found for device %s and interface %s", deviceID, interfaceID))
	}
	if *list.Payload.Count > 1 {
		return nil, fmt.Errorf(fmt.Sprintf("more than 1 ip found for device %s and interface %s", deviceID, interfaceID))
	}

	return list.Payload.Results[0], nil

}

// IPAddress retrieves the IPAddress by its ID
func (nb *Netbox) IPAddress(id int64) (*models.IPAddress, error) {
	params := ipam.NewIpamIPAddressesListParams()
	params.WithContext(context.Background())
	ids := fmt.Sprintf("%d", id)
	params.IDn = &ids
	limit := int64(1)
	params.Limit = &limit
	list, err := nb.client.Ipam.IpamIPAddressesList(params, nil)
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
		r, err := nb.Racks(role, strconv.FormatInt(s.ID, 10))
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

		r, err := nb.Servers(strconv.FormatInt(rack.ID, 10))
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
	params.WithContext(context.Background())
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

func client(host, token string) (*netboxclient.NetBoxAPI, error) {

	tlsClient, err := runtimeclient.TLSClient(runtimeclient.TLSClientOptions{InsecureSkipVerify: true})

	if err != nil {
		return nil, err
	}

	transport := runtimeclient.NewWithClient(host, netboxclient.DefaultBasePath, []string{"https"}, tlsClient)
	if token != "" {
		transport.DefaultAuthentication = runtimeclient.APIKeyAuth("Authorization", "header", fmt.Sprintf("Token %v", token))
	}

	c := netboxclient.New(transport, nil)

	return c, nil

}
