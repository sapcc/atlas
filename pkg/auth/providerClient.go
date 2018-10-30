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
