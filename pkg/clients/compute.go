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
	"sync"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type ComputeClient struct {
	*gophercloud.ServiceClient
	Authorized bool
}

func NewComputeClient(provider *gophercloud.ProviderClient) (*ComputeClient, error) {
	eo := gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic}

	sc, err := openstack.NewComputeV2(provider, eo)
	if err != nil {
		return nil, err
	}

	cc := &ComputeClient{
		ServiceClient: sc,
		Authorized:    true,
	}

	if err := checkAuthorized(sc); err != nil {
		cc.Authorized = false
	}

	return cc, nil
}

func checkAuthorized(sc *gophercloud.ServiceClient) error {
	_, err := servers.Get(sc, "dummy_test").Extract()
	switch err.(type) {
	case gophercloud.ErrUnexpectedResponseCode:
		// seems like our user misses the needed role!
		return err
	default:
		return nil
	}
}

func (c ComputeClient) GetServer(id string, sc chan<- *servers.Server, ec chan<- error, wg *sync.WaitGroup) {
	server, err := servers.Get(c.ServiceClient, id).Extract()
	defer wg.Done()
	if err != nil {
		ec <- err
	} else {
		sc <- server
	}
}
