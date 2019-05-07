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
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

type OSProvider struct {
	AuthURL           string `yaml:"auth_url"`
	User              string `yaml:"user"`
	Password          string `yaml:"password"`
	DomainName        string `yaml:"user_domain_name"`
	ProjectName       string `yaml:"project_name"`
	ProjectDomainName string `yaml:"domain_name"`
}

func NewProviderClient(pr OSProvider) (pc *gophercloud.ProviderClient, err error) {
	os.Setenv("OS_USERNAME", pr.User)
	os.Setenv("OS_PASSWORD", pr.Password)
	os.Setenv("OS_PROJECT_NAME", pr.ProjectName)
	os.Setenv("OS_DOMAIN_NAME", pr.DomainName)
	os.Setenv("OS_PROJECT_DOMAIN_NAME", pr.ProjectDomainName)
	os.Setenv("OS_AUTH_URL", pr.AuthURL)
	opts, err := openstack.AuthOptionsFromEnv()
	opts.AllowReauth = true
	opts.Scope = &gophercloud.AuthScope{
		ProjectName: opts.TenantName,
		DomainName:  os.Getenv("OS_PROJECT_DOMAIN_NAME"),
	}
	pc, err = openstack.AuthenticatedClient(opts)
	if err != nil {
		return pc, err
	}

	pc.UseTokenLock()

	return pc, nil
}
