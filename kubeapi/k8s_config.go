package kubeapi

import (
	"context"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var Config *rest.Config

func Init(initCtx context.Context, isDev bool) {
	var err error

	if isDev {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

		Config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	} else {
		Config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}
}
