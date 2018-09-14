/*******************************************************************************
*
* Copyright 2018 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

package clients

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type ComputeClient struct {
	*gophercloud.ServiceClient
}

func NewComputeClient(provider *gophercloud.ProviderClient) (*ComputeClient, error) {
	//serviceType := "compute"
	eo := gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic}
	//eo.ApplyDefaults(serviceType)

	sc, err := openstack.NewComputeV2(provider, eo)
	if err != nil {
		return nil, err
	}
	return &ComputeClient{
		ServiceClient: sc,
	}, nil
}

func (c ComputeClient) GetServer(id string) (*servers.Server, error) {
	server, err := servers.Get(c.ServiceClient, id).Extract()

	if err != nil {
		return server, err
	}

	return server, nil
}
