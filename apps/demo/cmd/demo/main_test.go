package main

import "testing"

func TestCanaryWeightAllowlist(t *testing.T) {
	for _, weight := range []int{0, 5, 25, 50, 100} {
		if !allowedCanaryWeight(weight) {
			t.Fatalf("weight %d", weight)
		}
	}
	for _, weight := range []int{-1, 1, 10, 75, 101} {
		if allowedCanaryWeight(weight) {
			t.Fatalf("accepted %d", weight)
		}
	}
}
