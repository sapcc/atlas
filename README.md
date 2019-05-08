# atlas

Atlas, brother of Prometheus.

This is a standalone custom Prometheus service discovery application. It consists of a growing number of discoveries which can be enabled/disabled and configured via a yaml file.

Available Discoveries:
1. Ironic nodes
2. Servers from netbox
3. Switches from netbox

## Architecture overview

![](https://github.com/sapcc/ipmi_sd/blob/master/documentation/ipmi_sd_arch.png)


## Install
A Dockerfile is provided to run it on Kubernetes. All necessary ENV VARs/flags can be figured out running `ipmi_sd --help`:

```
NAME:
   atlas - discovers custom services, enriches them with metadata labels and writes them to a file or Kubernetes configmap
 USAGE:
   atlas [global options]
 VERSION:
   0.1.7
 COMMANDS:
     help, h  Shows a list of commands or help for one command
 GLOBAL OPTIONS:
  - OS_PROM_CONFIGMAP_NAME: name of the configmap, where the discovered nodes should be written to.
  - K8S_NAMESPACE: name of the K8s namespace atlas is running in.
  - K8S_REGION: name of the k8s region atlas is running in
  - LOG_LEVEL: log level atlas should use:
    - "debug"
    - "error"
    - "warn"
    - "info"
```

To figure these out you can also just run it locally.
Either by building via docker using `docker build .` and then `docker run CONTAINER --help` or directly on bare metal if you have a working go
environment with `go run cmd/atlas/main.go --help`.

Prometheus server configuration:
A sample prometheus job to read those nodes is shown [here](https://github.com/sapcc/ipmi_sd/blob/master/prometheus.yml)
