# Provisioner

Creates and deletes the volumes on XenServer.

## Deployment

Create a configmap containing the *XenServer Host* you wish to provision SR PVs from.

```console
$ kubectl create configmap xenserver \
--from-literal=host=192.0.2.1
```

Create a secret containing the XenServer credentials. The credentials will be used by the provisioner and also the driver.

**WARNING:**
XenServer credentials will be passed on every node and persisted to disk for every volume.
At the moment this is the only way to provide XenServer storage support to Kubernetes.
The next release which will be build against Kubernetes 1.10 should however address this.

```console
$ kubectl create secret generic xenserver \
--from-literal=username=root \
--from-literal=password=Sup3rS3cretPassw0rd
```

```console
$ kubectl create -f deploy/deployment.yaml
deployment "xenserver-provisioner" created
```
If you are not using RBAC or OpenShift you can continue to the usage section.

### Authorization

If your cluster has RBAC enabled or you are running OpenShift you must authorize the provisioner.
If you are in a namespace/project other than "xenserver-provisioner" either edit `deploy/auth/clusterrolebinding.yaml` or edit the `oadm policy` command accordingly.

#### RBAC
```console
$ kubectl create -f deploy/auth/serviceaccount.yaml
serviceaccount "xenserver-provisioner" created
$ kubectl create -f deploy/auth/clusterrole.yaml
clusterrole "xenserver-provisioner-runner" created
$ kubectl create -f deploy/auth/clusterrolebinding.yaml
clusterrolebinding "run-xenserver-provisioner" created
$ kubectl patch deployment xenserver-provisioner -p '{"spec":{"template":{"spec":{"serviceAccount":"xenserver-provisioner"}}}}'
```

#### OpenShift
```console
$ oc create -f deploy/auth/serviceaccount.yaml
serviceaccount "xenserver-provisioner" created
$ oc create -f deploy/auth/openshift-clusterrole.yaml
clusterrole "xenserver-provisioner-runner" created
$ oadm policy add-cluster-role-to-user efs-provisioner-runner system:serviceaccount:xenserver-provisioner:xenserver-provisioner
$ oc patch deployment xenserver-provisioner -p '{"spec":{"template":{"spec":{"serviceAccount":"xenserver-provisioner"}}}}'
```

## Usage

First a [`StorageClass`](https://kubernetes.io/docs/user-guide/persistent-volumes/#storageclasses) for claims to ask for needs to be created.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: xenserver-default
parameters:
  spangenberg.io/xenserver/srName: Default
provisioner: spangenberg.io/xenserver-provisioner
```

### Parameters

* `spangenberg.io/xenserver/srName` : The name of the Storage Repository where XenServer should allocate the disks from.

Once you have finished configuring the class to have the name you chose when deploying the provisioner and the parameters you want, create it.

```console
$ kubectl create -f deploy/class.yaml 
storageclass "xenserver-default" created
```

When you create a claim that asks for the class, a volume will be automatically created.

```console
$ kubectl create -f deploy/claim.yaml 
persistentvolumeclaim "xenserver" created
$ kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS    CLAIM               STORAGECLASS        REASON    AGE
pvc-25de044e-5c93-11e8-885e-fe778d7189b9   1Mi        RWO            Delete           Bound     default/xenserver   xenserver-default             1s
```
