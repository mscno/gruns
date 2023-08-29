package main

import (
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"os"
)

func readJobs(file string) ([]job, error) {
	var root root
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Errorf("could not load deployment yml file: %s", file)
	}
	err = yaml.Unmarshal(bytes, &root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal yaml")
	}

	return root.Jobs, nil
}
