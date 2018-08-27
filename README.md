# ipmi_sd

A Prometheus service discovery to get the ipmi address of Openstack ironic nodes and enrich
them with nova metadata.

The discovery gets the ipmi addresses of all nodes via ironic and some metadata
from nova and writes these into a json in a Kubernetes configmap in a periodic interval.

The configmap can then be consumed by the Prometheus `file_sd` discovery
mechanism.
