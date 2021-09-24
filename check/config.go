package check

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/go-logr/logr"
	"github.com/ryanmt/cluster-resource-autoscaler/logging"
	v1 "k8s.io/api/core/v1"
)

// Support checks from CM
// Support checks from JSON

type ScalingTarget struct {
	Name      string
	Namespace string
	Type      string
}

type Spec struct {
	CPUPerReplica    float64 // In millicores
	MemoryPerReplica float64 // In bytes
	Name             string  // A defined name for this scaling configuration
	// ResourcePerReplica string        // How much resource per replica of the deployment
	Target ScalingTarget // What deployment to scale
	// TargetUtilization  float64       // Target utilization for the resourceName
}

// deploymentKey generates a unique string for comparing between Spec targets
func (s *ScalingTarget) deploymentKey() string {
	return fmt.Sprintf("%s/%s", s.Namespace, s.Name)
}

func (s *Spec) TargetKey() string {
	return s.Target.deploymentKey()
}

func (s *Spec) ResourceScaler(rName v1.ResourceName) float64 {
	switch rName {
	case v1.ResourceCPU:
		return s.CPUPerReplica
	case v1.ResourceMemory:
		return s.MemoryPerReplica
	}
	return 0.0
}

var logger logr.Logger

// Init configures our hooks for a logger
func Init(ctx context.Context) {
	logger = logging.FromContextOrDiscard(ctx)
}

func FromFile(jsonFile string) ([]Spec, error) {
	file, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}

	return FromReader(file)
}

func FromReader(r io.Reader) ([]Spec, error) {
	var specList []Spec
	var err error

	d := json.NewDecoder(r)

	// Read opening bracket
	_, err = d.Token()
	if err != nil {
		return specList, err
	}

	// TODO: Not sure if the the streaming version is a good idea in complexity...
	// just feels safer to model a bounded JSON implementation
	for d.More() {
		var s Spec
		err := d.Decode(&s)
		if err != nil {
			// Blah blah blah
			logger.Error(err, "Error decoding configuration")
			return specList, err
		}

		specList = append(specList, s)
	}
	_, err = d.Token() // Should just be the final bracket
	if err != nil {
		return specList, err
	}

	var depUnique map[string]bool

	// TODO: validate each check
	for _, s := range specList {
		if ok := depUnique[s.Target.deploymentKey()]; ok {
			// We've already seen this key...
			logger.V(-1).Info("Already have a configuration, skipping...", "deploymentKey", s.TargetKey())
		}
	}
	// if !validResourceName(s.ResourceName) {
	// 	fmt.Printf("Invalid resource name: %s\n", s.ResourceName)
	// }
	// }

	return specList, nil
}

// var validResourceNames = []string{"cpu", "memory", "ephemeral_storage", "huge_pages"}

// func validResourceName(name string) bool {
// 	for _, n := range validResourceNames {
// 		if n == name {
// 			return true
// 		}
// 	}

// 	return false
// }

// func lookupResourceByName(name string) corev1.ResourceName {
// 	switch name {
// 	case "cpu":
// 		return corev1.ResourceCPU
// 	case "memory":
// 		return corev1.ResourceMemory
// 	case "ephemeral_storage":
// 		return corev1.ResourceEphemeralStorage
// 	case "huge_pages":
// 		return corev1.ResourceHugePagesPrefix
// 	default:
// 		return corev1.ResourceCPU
// 	}
// }
