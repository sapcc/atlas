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
package auth

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/sapcc/ipmi_sd/pkg/config"
)

func NewProviderClient(opts config.Options) (p *gophercloud.ProviderClient, err error) {
	authOptions := gophercloud.AuthOptions{
		IdentityEndpoint: opts.IdentityEndpoint,
		Username:         opts.Username,
		Password:         opts.Password,
		DomainName:       opts.DomainName,
		AllowReauth:      true,
		Scope: &gophercloud.AuthScope{
			ProjectName: opts.ProjectName,
			DomainName:  opts.ProjectDomainName,
		},
	}
	p, err = openstack.AuthenticatedClient(authOptions)
	if err != nil {
		return p, err
	}

	p.UseTokenLock()

	return p, nil
}
