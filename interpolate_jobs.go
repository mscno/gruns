package main

import "strings"

func interpolateJobs(args args, jobs []job) []job {
	for i, j := range jobs {
		jobs[i].Image = interpolateString(args, j.Image)
		jobs[i].ServiceAccount = interpolateString(args, j.ServiceAccount)
	}
	return jobs
}

func interpolateString(args args, s string) string {
	s = strings.ReplaceAll(s, "${PROJECT_ID}", args.ProjectId)
	s = strings.ReplaceAll(s, "${PROJECT_NUMBER}", args.ProjectNumber)
	s = strings.ReplaceAll(s, "${REGION}", args.Region)
	s = strings.ReplaceAll(s, "${SERVICE_ACCOUNT}", args.ServiceAccount)
	s = strings.ReplaceAll(s, "${TRIGGER_SERVICE_ACCOUNT}", args.TriggerAccount)
	return s
}
