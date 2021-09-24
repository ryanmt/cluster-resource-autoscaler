package utilization

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/ryanmt/cluster-resource-autoscaler/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"

	"k8s.io/client-go/rest"
)

type MetricDatum struct {
	Timestamp time.Time
	Window    time.Duration
	Value     int64
}

type Metrics map[string]MetricDatum

var logger logr.Logger
var metricClient *resourceclient.MetricsV1beta1Client
var k8client *kubernetes.Clientset
var ctx context.Context

func Init(initCtx context.Context, isDev bool) {
	var err error
	var config *rest.Config

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

	metricClient = resourceclient.NewForConfigOrDie(config)
	k8client = kubernetes.NewForConfigOrDie(config)
}

// CapacityByResource current cluster capacity of given resource in cores or kilobytes
func CapacityByResource(resource corev1.ResourceName) int64 {
	n, err := k8client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var allocatableResource int64

	for _, node := range n.Items {
		quantity := node.Status.Allocatable[resource]
		allocatableResource += quantity.MilliValue()
	}

	logger.V(2).Info("Node resources allocated", "resource", resource, "value", allocatableResource)
	return allocatableResource / 1000
}

func UtilizationByResource(resource corev1.ResourceName) int64 {
	nodeMetrics, err := metricClient.NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting Metrics: %v\n", err.Error())
	}

	nMetrics := getMetrics(nodeMetrics.Items, resource)

	var nodeResourceUsage int64

	for k, m := range nMetrics {
		logger.V(3).Info("Found node metric", "resource", resource.String(), "node", k, "value", m.Value)
		nodeResourceUsage += m.Value
	}
	logger.V(2).Info("Node utilization", "resource", resource, "value", nodeResourceUsage)

	return nodeResourceUsage
}

func PercentageByResource(resource corev1.ResourceName) float64 {
	return float64(UtilizationByResource(resource)) / float64(CapacityByResource(resource))
}

// resourceNames
type CoreResourceNames = []corev1.ResourceName

func ResourceNames() []corev1.ResourceName {
	return []corev1.ResourceName{
		corev1.ResourceCPU,
		corev1.ResourceEphemeralStorage,
		corev1.ResourceHugePagesPrefix,
		corev1.ResourceMemory,
		corev1.ResourceStorage,
	}
}

func getMetrics(rawNodeMetrics []v1beta1.NodeMetrics, resource corev1.ResourceName) Metrics {
	res := make(Metrics, len(rawNodeMetrics))
	for _, m := range rawNodeMetrics {
		resValue, found := m.Usage[resource]
		if !found {
			logger.V(2).Info("Missing resource metric", "resourceName", resource.String(), "namespace", m.Namespace, "name", m.Name)
			break
		}
		res[m.Name] = MetricDatum{
			Timestamp: m.Timestamp.Time,
			Window:    m.Window.Duration,
			Value:     resValue.Value(),
		}
	}
	return res
}
