## namespace-scc-operator
This operator will create a pre-defined SCC with a few configurable fields for every namespace that is not present in the whitelist.  
The SCCs will be named `<namespacescc>-<namespace>`  
Multiple `namespacesccs` can be created with different values. They will result in new SCCs being created for each namespace not on the whitelist.  
The owner of the created SCC is the corresponding namespace. An SCC gets garbage collected if the namespace is deleted.
```yaml 
apiVersion: namespacescc.github.com/v1alpha1
kind: NamespaceSCC
metadata:
  name: test-scc
spec:
  uuid: 123123123
  sccPriority: 55
  whiteList:
  - "openshift-apiserver"
  - "openshift-console"
```
`uuid`: The UUID the SCC will be configured with  
`sccPriority`: the priority the SCC will be configured with  
`whiteList`: no SCC will be created for the namespaces in the list  
## Prerequisites  
* docker/podman
* [operator-sdk v0.10.1](https://github.com/operator-framework/operator-sdk/releases/tag/v0.10.1)
## Build
```bash 
cd namespace-scc-operator  
operator-sdk build <image-repo>:<tag> [--image-builder podman] [--verbose]  
{podman|docker} push <image-repo>:<tag>
```
## Deploy
```bash 
cd namespace-scc-operator
oc create -f deploy/service_account.yaml
oc create -f deploy/clusterrole.yaml
oc create -f deploy/clusterrolebinding.yaml
oc create -f deploy/crds/namespacescc_v1alpha1_namespacescc_crd.yaml
sed -i 's/REPLACE_IMAGE/<image-repo>:<tag>/g' deploy/operator.yaml
oc create -f deploy/operator.yaml
```
## Use
```bash 
oc create -f deploy/crds/namespacescc_v1alpha1_namespacescc_cr.yaml
```
## SCC
```yaml
allowHostDirVolumePlugin: false          
allowHostNetwork: false             
allowHostPID: false                        
allowHostPorts: false     
allowPrivilegeEscalation: true
allowPrivilegedContainer: false
allowedCapabilities: null
apiVersion: security.openshift.io/v1
defaultAddCapabilities: null
fsGroup:
  ranges:
  - max: <uuid>
    min: <uuid>
  type: MustRunAs
groups:
- mapr-sas
kind: SecurityContextConstraints
metadata:
  labels:
    namespace: <namespace>
  name: <namespacescc>-<namespace>
  ownerReferences:
  - apiVersion: v1
    blockOwnerDeletion: true
    controller: true
    kind: Namespace
    name: <namespace>
priority: <sccPriority>
readOnlyRootFilesystem: false
requiredDropCapabilities:
- KILL
- MKNOD
- SETUID
- SETGID
runAsUser:
  type: MustRunAs
  uid: <uuid>
seLinuxContext:
  type: MustRunAs
supplementalGroups:
  ranges:
  - max: <uuid>
    min: <uuid>
  type: RunAsAny
users:
- system:serviceaccount:<project>:default
volumes:
- configMap
- downwardAPI
- emptyDir
- persistentVolumeClaim
- projected
- secret
```