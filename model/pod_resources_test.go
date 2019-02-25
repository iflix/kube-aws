package model

import (
	"testing"
)

func TestInvalidCpu(t *testing.T) {
	res := PodResources{Cpu: "12", Memory: "100Mi"}
	if res.IsValid() {
		t.Errorf("Expected the test to fail as the cpu is invalid.")
	}

	res = PodResources{Cpu: "abdbsd", Memory: "100Mi"}
	if res.IsValid() {
		t.Errorf("Expected the test to fail as the cpu is invalid.")
	}

	res = PodResources{Cpu: "30b", Memory: "100Mi"}
	if res.IsValid() {
		t.Errorf("Expected the test to fail as the cpu is invalid.")
	}
}

func TestInvalidMemory(t *testing.T) {
	res := PodResources{Cpu: "100m", Memory: "asdba"}
	if res.IsValid() {
		t.Errorf("Expected the test to fail as the memory is invalid.")
	}

	res = PodResources{Cpu: "100m", Memory: "100m"}
	if res.IsValid() {
		t.Errorf("Expected the test to fail as the memory is invalid.")
	}

	res = PodResources{Cpu: "100m", Memory: "50Mid"}
	if res.IsValid() {
		t.Errorf("Expected the test to fail as the memory is invalid.")
	}
}

func TestValidRequests(t *testing.T) {
	res := PodResources{Cpu: "100m", Memory: "20Mi"}
	if !res.IsValid() {
		t.Errorf("Expected the test to fail as the memory is invalid.")
	}

	res = PodResources{Cpu: "5m", Memory: "500Gi"}
	if !res.IsValid() {
		t.Errorf("Expected the test to fail as the memory is invalid.")
	}
}
