package main

import "github.com/pkg/errors"

func validateJob(j job) error {
	if j.Image == "" {
		return errors.New("image cannot be empty")
	}
	//if j.Schedule == "" {
	//	return errors.New("schedule cannot be empty")
	//}

	//if j.Args == "" {
	//	return errors.New("args cannot be empty")
	//}

	return nil
}
