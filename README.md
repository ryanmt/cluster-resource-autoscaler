# Cluster Resource Autoscaler

Scale something in proportion to the scale of the cluster compute resources.  This might make more sense in
some cases than pod/node count based scaling as the load carried by a pod or node could be drastically
distinct.

## MVP

This is currently very much an MVP implementation.  Some ideas about how to improve this further or expand the
scope are:
- Support scaling on arbitrary resource types (since those are themselves extensible in the k8s API) rather
  than just CPU/memory
- Allow for multiple scaling definitions against a single `Target` deployment/replicaset/statefulset
- HA leader election for resiliency of this autoscaler
- Histeresis in scaling to avoid flapping
- Support any "Scalable" API entity rather than just deployments, replicasets, and statefulsets.

## Configuring a target for autoscaling

Please mount a configmap containing a valid `config.json` JSON configuration key into the deployment of this
application.  CRA will automatically read any changes to the configuration on the *next* tick of its update
loop.

### Configuration Schema

See example in `test_config.json`.

`config.json` object is an array of individual checks, each of which should specify a particular target for
scaling.

```json
[
  {
    "MemoryPerReplica": 100e9,
    "CPUPerReplica": 16,
    "Target": {
      "Name": "scalable-service",
      "Namespace": "default",
      "Type": "deployment"
    }
  }
]
```

| *Key* | *Description* |
| ---- | ----------- |
| *MemoryPerReplica* | The amount of memory to target for each replica, expressed in bytes |
| *CPUPerReplica* | The number of cores to target for each replica, expressed in cores |
| *Target.Name* | The name of the "target" "kind" which should be selected to scale |
| *Target.Namespace* | The namespace of the "target" to scale |
| *Target.Type* | The kind of object which we are scaling.  Must be a member of `{deployment,replicaset,statefulset}` |


## Deploying

Please see the example in [manifests/all.yaml](./manifests/all.yaml).

### ClusterRole
Must provide a `ClusterRole` which allows:

```yaml
- apiGroups:
  - ""
  resources:
  - nodes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metrics.k8s.io
  resources:
  - nodes
  verbs:
  - get
  - list
- apiGroups:
  - apps
  resources:
  - deployments/scale
  - replicasets/scale
  - statefulsets/scale
  verbs:
  - get
  - patch
```

Node permissions are required to determine how much cluster compute is available

Metric permissions are only now required for debugging logging statements and will be removed in a future version

*kind*/scale permissions are required to check current scale and to apply scale updates to targets

### Development

```export RESOURCE_AUTOSCALER_TESTING_MODE=yes ; inotifyrun go run ./main.go -- -v=9 --logging-format=json```

