package utils

import "testing"

func TestCheckApiVersion(t *testing.T) {
	validVersions := []string{
		"1.0.0",
		"2.3.4",
		"10.20.30",
	}
	for _, version := range validVersions {
		if !CheckApiVersion(version) {
			t.Errorf("Expected version %s to be valid, but it was invalid", version)
		}
	}

	invalidVersions := []string{
		"1.0",
		"2.3.4.5",
		"10.20",
		"abc",
	}
	for _, version := range invalidVersions {
		if CheckApiVersion(version) {
			t.Errorf("Expected version %s to be invalid, but it was valid", version)
		}
	}
}
