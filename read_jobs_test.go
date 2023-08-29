package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_ReadJobs(t *testing.T) {
	jobs, err := readJobs("data/test.yml")
	require.NoError(t, err)
	require.Equal(t, 2, len(jobs))
	require.Equal(t, "test", jobs[0].Name)
	require.Equal(t, "test", jobs[0].Image)
	require.Equal(t, "test", jobs[0].Schedule)
	require.Equal(t, "test", jobs[0].Args)

	_, err = readJobs("data/does_not_exist.yml")
	require.Error(t, err)
}
