module github.com/ryanmt/cluster-resource-autoscaler

go 1.17

require k8s.io/client-go v0.22.2

require (
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/sys v0.0.0-20210917161153-d61c044b1678 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/metrics v0.22.2
)

require (
	github.com/go-logr/logr v0.4.0
	k8s.io/klog/v2 v2.9.0
)
