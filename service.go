package main

import (
	run "cloud.google.com/go/run/apiv2"
	scheduler "cloud.google.com/go/scheduler/apiv1"
	"context"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

type service struct {
	jobclient             *run.JobsClient
	cscclient             *scheduler.CloudSchedulerClient
	project               string
	region                string
	defaultServiceAccount string
	defaultTriggerAccount string
	disableTriggers       bool
}

func initializeService(ctx context.Context, args args) *service {
	cscclient, err := scheduler.NewCloudSchedulerClient(ctx, option.WithScopes("https://www.googleapis.com/auth/cloud-platform"))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	jobclient, err := run.NewJobsClient(ctx, option.WithScopes("https://www.googleapis.com/auth/cloud-platform"))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	return &service{
		cscclient:             cscclient,
		jobclient:             jobclient,
		project:               args.ProjectId,
		region:                args.Region,
		defaultServiceAccount: args.ServiceAccount,
		defaultTriggerAccount: args.TriggerAccount,
		disableTriggers:       args.DisableTriggers,
	}
}
