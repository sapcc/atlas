package netbox

import (
	"fmt"
	"log"
	"strings"

	runtimeclient "github.com/go-openapi/runtime/client"
	netboxclient "github.com/hosting-de-labs/go-netbox/netbox/client"
	"github.com/hosting-de-labs/go-netbox/netbox/client/dcim"
	"github.com/hosting-de-labs/go-netbox/netbox/models"
)

const netboxDefaultHost = "netbox.global.cloud.sap"

type Netbox struct {
	client *netboxclient.NetBox
}

func NewDefaultHost(token string) (*Netbox, error) {
	return New(netboxDefaultHost, token)
}
func New(host, token string) (*Netbox, error) {
	client, err := client(host, token)
	if err != nil {
		return nil, err
	}
	return &Netbox{client: client}, nil
}
//TODO:onur document functions
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
func (nb *Netbox) Servers(rackID int64) ([]models.Device, error) {
	result := make([]models.Device, 0)
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
func (nb *Netbox) RacksByRegion(role string, region string) ([]models.Rack, error) {

	siteResults, err := nb.Sites("qa-de-1")
	if err != nil {
		log.Fatal(err)
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
func (nb *Netbox) ServersByRegion(rackRole string, region string) ([]models.Device, error) {
	racks, err := nb.RacksByRegion(rackRole, region)
	if err != nil {
		log.Fatal(err)
	}

	results := make([]models.Device, 0)

	for _, rack := range racks {

		r, err := nb.Servers(rack.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, r...)
	}

	return results, nil
}
func (nb *Netbox) RacksByRegionNameCheck(role string, region string) ([]models.Rack, error) {
	allRacks, err := nb.Racks(role, 0)
	if err != nil {
		return nil, err
	}
	result := make([]models.Rack, 0)

	for _, rack := range allRacks {
		if rack.Site != nil && rack.Site.Name != nil && caseInsensitiveContains(*rack.Site.Name, region) {
			result = append(result, rack)
		}
	}

	return result, nil
}
func caseInsensitiveContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
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