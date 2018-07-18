# ipmi_sd

Custom Prometheus service discovery to get ipmi address of ironic nodes.

Discovery service gets ipmi addresses of all nodes via ironic and writes those targets into a configmap every x seconds.
