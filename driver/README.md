# Driver

Mounts and unmounts the volumes on the hosts itself.

## Installing

The volume driver needs to be installed on all nodes.

### Kubernetes

Kubernetes can install the driver through a *DaemonSet*, which can be created
by running this command:

```console
$ kubectl apply -f deploy/daemon.yaml
```

### OpenShift

OpenShift doesn't allow installing the driver trough a *DaemonSet*, execute the
the following commands on each node:

```console
mkdir -p /usr/libexec/kubernetes/kubelet-plugins/volume/exec/spangenberg.io~xenserver && \
wget https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 -O /usr/libexec/kubernetes/kubelet-plugins/volume/exec/spangenberg.io~xenserver/jq && \
chmod +x /usr/libexec/kubernetes/kubelet-plugins/volume/exec/spangenberg.io~xenserver/jq && \
wget https://raw.githubusercontent.com/spangenberg/kubernetes-xenserver-storage/v0.1.0/driver/xenserver -O /usr/libexec/kubernetes/kubelet-plugins/volume/exec/spangenberg.io~xenserver/xenserver && \
chmod +x /usr/libexec/kubernetes/kubelet-plugins/volume/exec/spangenberg.io~xenserver/xenserver
```
