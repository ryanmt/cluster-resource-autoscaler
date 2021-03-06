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

var logger logr.Logger

// Init configures our hooks for a logger
func Init(ctx context.Context) {
	logger = logging.FromContextOrDiscard(ctx)
}

// Support checks from CM :done:
// Support checks from JSON :done:
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

	depUnique := make(map[string]bool)

	// TODO: Not sure if the the streaming version is a good idea in complexity...
	// just feels safer to model a bounded JSON parsing implementation
	for d.More() {
		var s Spec
		err := d.Decode(&s)
		if err != nil {
			// Blah blah blah
			logger.Error(err, "Error decoding configuration")
			return specList, err
		}

		if _, ok := depUnique[s.TargetKey()]; ok {
			// We've already seen this key which is a bad configuration.  Probs should error or something but RN this is just a info statement. :|
			logger.Info("Already have a configuration, skipping...", "deploymentKey", s.TargetKey(), "checkSpec", s.Name)
			continue
		} else {
			depUnique[s.TargetKey()] = true
		}
		specList = append(specList, s)
	}
	_, err = d.Token() // Should just be the final bracket
	if err != nil {
		return specList, err
	}

	return specList, nil
}

type ScalingTarget struct {
	Name      string
	Namespace string
	Kind      string
}

// Key generates a unique string for comparing between Spec targets
func (s *ScalingTarget) Key() string {
	return fmt.Sprintf("%s->%s/%s", s.Kind, s.Namespace, s.Name)
}

type Spec struct {
	CPUPerReplica    float64 // In millicores
	MemoryPerReplica float64 // In bytes
	Name             string  // A defined name for this scaling configuration
	// ResourcePerReplica string        // How much resource per replica of the deployment
	Target ScalingTarget // What deployment to scale
	// TargetUtilization  float64       // Target utilization for the resourceName
}

func (s *Spec) TargetKey() string {
	return s.Target.Key()
}

func (s *Spec) ResourceScaler(rName v1.ResourceName) float64 {
	// TODO: Add ability to scale on these components as well:
	// corev1.ResourceEphemeralStorage
	// corev1.ResourceHugePagesPrefix
	switch rName {
	case v1.ResourceCPU:
		return s.CPUPerReplica
	case v1.ResourceMemory:
		return s.MemoryPerReplica
	}
	return 0.0
}
