package check_test

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/ryanmt/cluster-resource-autoscaler/check"
	corev1 "k8s.io/api/core/v1"
)

func GiveMeAConfigSpecFile() string {
	f, _ := os.CreateTemp("", "")

	specs := []check.Spec{
		{ResourceName: "cpu", TargetUtilization: 0.99},
	}

	b, _ := json.Marshal(specs)

	f.Write(b)
	f.Close()

	return f.Name()
}

func TestSpec_Resource(t *testing.T) {
	type fields struct {
		ResourceName      string
		TargetUtilization float64
		Scaler            string
	}
	tests := []struct {
		name   string
		fields fields
		want   corev1.ResourceName
	}{
		{"cpu", fields{ResourceName: "cpu"}, corev1.ResourceCPU},
		{"ephemeral_storage", fields{ResourceName: "ephemeral_storage"}, corev1.ResourceEphemeralStorage},
		{"huge_pages", fields{ResourceName: "huge_pages"}, corev1.ResourceHugePagesPrefix},
		{"memory", fields{ResourceName: "memory"}, corev1.ResourceMemory},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &check.Spec{
				ResourceName:      tt.fields.ResourceName,
				TargetUtilization: tt.fields.TargetUtilization,
				Scaler:            tt.fields.Scaler,
			}
			if got := s.Resource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Spec.Resource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromFile(t *testing.T) {
	tests := []struct {
		name    string
		want    []check.Spec
		wantErr bool
	}{
		{"happy path", []check.Spec{{ResourceName: "cpu", TargetUtilization: 0.99}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := GiveMeAConfigSpecFile()

			got, err := check.FromFile(filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromReader(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []check.Spec
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := check.FromReader(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromReader() = %v, want %v", got, tt.want)
			}
		})
	}
}
