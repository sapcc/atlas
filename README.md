# ipmi service discovery

This is a standalone custom Prometheus service discovery applicationto for ironic nodes.
Discovery service gets ipmi addresses of all nodes via ironic and writes those targets into a configmap every x seconds.

## Architecture overview

![](https://github.com/sapcc/ipmi_sd/blob/master/documentation/ipmi_sd_arch.png)


## Install
A Dockerfile is provided to run it on Kubernetes. The following environment variables are needed:
- OS_PROM_CONFIGMAP_NAME: name of the configmap, where the discovered nodes should be written to.
- REFRESH_INTERVAL: interval in seconds, the prozess to look for new ironic nodes should be run. (Default: 600s)
- Openstack auth:
  - OS_AUTH_URL
  - OS_USERNAME
  - OS_PASSWORD
  - OS_USER_DOMAIN_NAME
  - OS_PROJECT_NAME
  - OS_PROJECT_DOMAIN_NAME


Prometheus server configuration:
A sample prometheus job to read those nodes is shown [here](https://github.com/sapcc/ipmi_sd/blob/master/prometheus.yml)
