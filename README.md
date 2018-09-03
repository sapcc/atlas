# ipmi service discovery

This is a standalone custom Prometheus service discovery applicationto for ironic nodes.
Discovery service gets ipmi addresses of all nodes via ironic and writes those targets into a configmap every x seconds.

## Architecture overview

![](https://github.com/sapcc/ipmi_sd/blob/master/documentation/ipmi_sd_arch.png)


## Install

A Dockerfile is provided to run it on Kubernetes.
All necessary ENV VARs/flags can be figured out running `ipmi_sd --help`:

```
NAME:
   ipmi_sd - discover OpenStack Ironic nodes for Prometheus, enrich them with metadata labels from Nova and write them to a file or Kubernetes configmap

USAGE:
   ipmi_sd [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --auth-url value        OpenStack identity endpoint URI [$OS_AUTH_URL]
   --configmap             Write file into a Kubernetes configmap
   --configmap-name value  Name of the configmap to be created (default: "ipmi-sd") [$OS_PROM_CONFIGMAP_NAME]
   --debug                 Enable more verbose logging
   --filename value        Output file name for the file_sd compatible file. (default: "ipmi_targets.json")
   --interval value        Refresh interval for fetching ironic nodes (default: 600) [$REFRESH_INTERVAL]
   --password value        The OpenStack password. Declaration by flag is inherently insecure, because every user can read flags of running programs [$OS_PASSWORD]
   --project value         OpenStack project name [$OS_PROJECT_NAME]
   --project-domain value  OpenStack project domain name [$OS_PROJECT_DOMAIN_NAME]
   --user value            OpenStack username [$OS_USERNAME]
   --user-domain value     OpenStack user domain name [$OS_USER_DOMAIN_NAME]
   --help, -h              show help
   --version, -v           print the version
```

To figure these out you can also just run it locally.
Either by building via docker using `docker build .` and then `docker run CONTAINER --help` or directly on bare metal if you have a working go
environment with `go run cmd/discovery/ipmi_discovery.go --help`.

Prometheus server configuration:
A sample prometheus job to read those nodes is shown [here](https://github.com/sapcc/ipmi_sd/blob/master/prometheus.yml)
