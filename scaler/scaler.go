package scaler

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/ryanmt/cluster-resource-autoscaler/check"
	"github.com/ryanmt/cluster-resource-autoscaler/logging"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var ctx context.Context
var k8client *kubernetes.Clientset
var logger logr.Logger
var scaler scale.ScalesGetter

func Init(initCtx context.Context, isDev bool) {
	var config *rest.Config
	var err error

	logger = logging.FromContextOrDiscard(initCtx)
	ctx = initCtx

	if isDev {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	cachedDiscoveryClient := cacheddiscovery.NewMemCacheClient(discoveryClient)

	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discoveryClient)
	scaler, err = scale.NewForConfig(config, restMapper, dynamic.LegacyAPIPathResolverFunc, scaleKindResolver)
	if err != nil {
		panic(err.Error())
	}

	k8client = kubernetes.NewForConfigOrDie(config)
}

func GetReplicas(target check.ScalingTarget) (int32, error) {
	gr := lookupGroupResource(target)
	currentScale, err := scaler.Scales(target.Namespace).Get(ctx, gr, target.Name, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	logger.Info("Current scale", "scale", currentScale.Status.Replicas)

	return currentScale.Status.Replicas, nil
}

func UpdateReplicas(target check.ScalingTarget, desiredReplicas int32) (prevReplicas int32, err error) {
	gr := lookupGroupResource(target)

	s := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: desiredReplicas,
		},
	}

	newScale, err := scaler.Scales(target.Namespace).Update(ctx, gr, s, metav1.UpdateOptions{})
	return newScale.Status.Replicas, err
}

func lookupGroupResource(target check.ScalingTarget) schema.GroupResource {
	var group string = "apps"
	// switch target.Type {
	// case "deployment":
	//   group = "apps"
	// case "replicaset":
	//   group = "apps"
	// case "statefulset":
	//   group = "apps"
	// }

	logger.Info("ScalingTarget", "target", target)
	thing := fmt.Sprintf("%s.%s", target.Type, group)
	logger.Info("GroupResource is weird", "gr", thing)
	return schema.ParseGroupResource(thing)
}
