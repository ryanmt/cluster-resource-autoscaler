package kubeapi

import resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"

func MetricClient() *resourceclient.MetricsV1beta1Client {
	return resourceclient.NewForConfigOrDie(Config)
}
