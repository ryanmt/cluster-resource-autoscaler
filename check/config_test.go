package check_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ryanmt/cluster-resource-autoscaler/check"
	v1 "k8s.io/api/core/v1"
)

func GiveMeAConfigSpecFile() string {
	f, _ := os.CreateTemp("", "")

	fakeTarget := check.ScalingTarget{
		Name:      "name",
		Namespace: "default",
		Kind:      "deployment",
	}

	specs := []check.Spec{
		{Name: "cpu", CPUPerReplica: 12, Target: fakeTarget},
	}

	b, _ := json.Marshal(specs)

	f.Write(b)
	f.Close()

	return f.Name()
}

func GiveMeATarget() check.ScalingTarget {
	return check.ScalingTarget{
		Name:      "name",
		Namespace: "default",
		Kind:      "deployment",
	}
}

func GiveMeASpec() check.Spec {
	return check.Spec{Name: "cpu", CPUPerReplica: 12, MemoryPerReplica: 12, Target: GiveMeATarget()}
}

func TestSpec_ResourceScaler(t *testing.T) {
	tests := []struct {
		name          string
		specification check.Spec
	}{
		{"good one", GiveMeASpec()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rCPU := check.SupportedResources()[0]
			rMem := check.SupportedResources()[1]
			if tt.specification.ResourceScaler(rCPU) != tt.specification.CPUPerReplica {
				t.Errorf("check.Spec.ResourceScaler() = %v, want %v", tt.specification.ResourceScaler(rCPU), tt.specification.CPUPerReplica)
			}
			if tt.specification.ResourceScaler(rMem) != tt.specification.MemoryPerReplica {
				t.Errorf("check.Spec.ResourceScaler() = %v, want %v", tt.specification.ResourceScaler(rMem), tt.specification.MemoryPerReplica)
			}
		})
	}

	// Test the edge case clause
	var nullValue float64
	spec := tests[0].specification
	if spec.ResourceScaler(v1.ResourceHugePagesPrefix) != nullValue {
		t.Errorf("check.Spec.ResourceScaler() should default to 0.0; got %v", spec.ResourceScaler(v1.ResourceHugePagesPrefix))
	}
}

func TestFromReader(t *testing.T) {
	tests := []struct {
		name     string
		expected []check.Spec
		wantErr  bool
	}{
		{"happy path", []check.Spec{GiveMeASpec()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var file bytes.Buffer
			bytes, err := json.Marshal(tt.expected)
			if err != nil {
				t.Fatal(err)
			}
			file.Write(bytes)

			got, err := check.FromReader(strings.NewReader(string(bytes)))
			if (err != nil) != tt.wantErr {
				t.Errorf("FromReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("FromReader() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScalingTarget_Key(t *testing.T) {
	type fields struct {
		Name      string
		Namespace string
		Kind      string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &check.ScalingTarget{
				Name:      tt.fields.Name,
				Namespace: tt.fields.Namespace,
				Kind:      tt.fields.Kind,
			}
			if got := s.Key(); got != tt.want {
				t.Errorf("ScalingTarget.Key() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpec_TargetKey(t *testing.T) {
	type fields struct {
		CPUPerReplica    float64
		MemoryPerReplica float64
		Name             string
		Target           check.ScalingTarget
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &check.Spec{
				CPUPerReplica:    tt.fields.CPUPerReplica,
				MemoryPerReplica: tt.fields.MemoryPerReplica,
				Name:             tt.fields.Name,
				Target:           tt.fields.Target,
			}
			if got := s.TargetKey(); got != tt.want {
				t.Errorf("check.Spec.TargetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
