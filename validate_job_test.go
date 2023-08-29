package main

import "testing"

func TestValidateJob(t *testing.T) {
	j := job{
		Image:    "test",
		Schedule: "test",
		Args:     "test",
	}

	err := validateJob(j)
	if err != nil {
		t.Error("validateJob should not return an error")
	}
}
