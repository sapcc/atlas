# atlas

Atlas, brother of Prometheus.

This is a standalone custom Prometheus service discovery application. It consists of a growing number of discoveries which can be enabled/disabled and configured via a yaml file. The discovered targets are then written into a configmap file, which is consumed by Prometheus.

## Architecture overview

![](https://github.com/sapcc/ipmi_sd/blob/master/documentation/ipmi_sd_arch.png)


Available Discoveries:
1. Ironic Nodes
```
discoveries:
      ironic:
        refresh_interval: 600 #How often the discovery should check for new/updated nodes.
        targets_file_name: "ironic.json" #Name of the file to write the nodes to.
        os_auth: # Openstack auth
          auth_url: openstack auth url
          user: openstack user
          password: os user pw
          user_domain_name: openstack user_domain_name
          project_name: openstack project_name
          domain_name: openstack domain_name
```
2. Netbox API
  - DCIM-Devices
    ```
    netbox:
        refresh_interval: 600 # How often the discovery should check for new/updated devices.
        targets_file_name: "netbox.json"  #Name of the file to write the devices to.
        netbox_host: "netbox_host_url"
        netbox_api_token: "netbox_api_token"
        dcim:
          devices: #Array of device queries
            - custom_labels: #Use to add custom labels to the target
                anyLabel: "anyValue"
                job: "job_name"
              target: 1 #Query Parameters: Any parameters the netbox api ([netbox_url]/api/dcim/devices/) accepts.
              role: "role_name"
              manufacturer: "cisco"
              region: "de1"
              status: "1"
            - custom_labels: ....
    ```
  - Virtualization-VMs
    ```
    netbox:
        refresh_interval: 600 # How often the discovery should check for new/updated devices.
        targets_file_name: "netbox.json"  #Name of the file to write the devices to.
        netbox_host: "netbox_host_url"
        netbox_api_token: "netbox_api_token"
        virtualization:
          vm: #Array of vms queries
            - custom_labels: #Use to add custom labels to the target
                anyLabel: "anyValue"
                job: "job_name"
              target: 1 #Query Parameters: Any parameters the netbox api ([netbox_url]/api/virtualization/virtual-machines/",) accepts.
              manufacturer: "cisco"
              region: "de1"
              tag: "tag_name"
            - custom_labels: ....
    ```

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
A sample prometheus job to read those configmap targets is shown [here](https://github.com/sapcc/ipmi_sd/blob/master/prometheus.yml)