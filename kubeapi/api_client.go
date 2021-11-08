package kubeapi

import "k8s.io/client-go/kubernetes"

func APIClient() *kubernetes.Clientset {
	return kubernetes.NewForConfigOrDie(Config)
}
