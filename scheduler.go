package main

import (
	"cloud.google.com/go/scheduler/apiv1/schedulerpb"
	"context"
	"fmt"
	"github.com/elliotchance/pie/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
)

func deleteSchedulerJobs(c *service, validTriggerNames []string) error {
	ctx := context.Background()
	iter := c.cscclient.ListJobs(ctx, &schedulerpb.ListJobsRequest{
		Parent:    c.parent(),
		PageSize:  500,
		PageToken: "",
	})

	for {
		res, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return err
		}

		if !pie.Contains(validTriggerNames, trimParent(c.parent(), res.Name)) {
			log.Debug().Msgf("deleting trigger %s ", res.Name)
			err := c.cscclient.DeleteJob(ctx, &schedulerpb.DeleteJobRequest{Name: res.Name})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func handleSchedulerJob(c *service, j job) error {
	ctx := context.Background()
	var update bool
	uri := triggerUri(c.project, c.region, j.Name)

	scheduledJob, err := c.getOrCreateSchedulerJob(ctx, j)
	if err != nil {
		return errors.Wrap(err, "get or create scheduler job failed")
	}
	if scheduledJob.State == schedulerpb.Job_PAUSED && !c.disableTriggers {
		log.Debug().Msgf("Enabling trigger: %s", scheduledJob.Name)
		scheduledJob.State = schedulerpb.Job_ENABLED
		_, err := c.cscclient.ResumeJob(ctx, &schedulerpb.ResumeJobRequest{Name: scheduledJob.Name})
		if err != nil {
			return err
		}
	} else if scheduledJob.State == schedulerpb.Job_ENABLED && c.disableTriggers {
		log.Debug().Msgf("Disabling trigger: %s", scheduledJob.Name)
		scheduledJob.State = schedulerpb.Job_PAUSED
		_, err := c.cscclient.PauseJob(ctx, &schedulerpb.PauseJobRequest{Name: scheduledJob.Name})
		if err != nil {
			return err
		}
	}

	if scheduledJob.Schedule != j.Schedule {
		log.Debug().Msgf("Updating schedule for trigger %s from %s to %s", scheduledJob.Name, scheduledJob.Schedule, j.Schedule)
		scheduledJob.Schedule = j.Schedule
		update = true
	}

	if scheduledJob.TimeZone != defaultTimezone {
		log.Debug().Msgf("Updating timezone for trigger %s from %s to %s", scheduledJob.Name, scheduledJob.TimeZone, defaultTimezone)
		scheduledJob.TimeZone = defaultTimezone
		update = true
	}

	target, ok := scheduledJob.Target.(*schedulerpb.Job_HttpTarget)
	if !ok {
		return errors.Errorf("bad target for trigger %s", scheduledJob.Name)
	}

	if target.HttpTarget.Uri != uri {
		log.Debug().Msgf("Updating url for trigger %s from %s to %s", scheduledJob.Name, target.HttpTarget.Uri, uri)
		scheduledJob.Target = targetFromUri(c.defaultServiceAccount, uri)
		update = true
	}

	if update {
		log.Debug().Msgf("Updating trigger: %s - %s", scheduledJob.Name, scheduledJob.State)
		_, err := c.cscclient.UpdateJob(ctx, &schedulerpb.UpdateJobRequest{
			Job: scheduledJob,
			//UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"schedule", "http_target.uri", "time_zone", "state"}},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func getSchedulerResourceName(project, region, jobName string) string {
	return fmt.Sprintf("projects/%s/locations/%s/jobs/%s-trigger", project, region, jobName)
}

func (c *service) getOrCreateSchedulerJob(ctx context.Context, j job) (*schedulerpb.Job, error) {
	job, err := c.getSchedulerJob(ctx, j.Name)
	if err != nil {
		return c.createSchedulerJob(ctx, j)
	}
	return job, nil
}

func (c *service) getSchedulerJob(ctx context.Context, jobName string) (*schedulerpb.Job, error) {
	name := getSchedulerResourceName(c.project, c.region, jobName)
	return c.cscclient.GetJob(ctx, &schedulerpb.GetJobRequest{Name: name})
}

func (c *service) createSchedulerJob(ctx context.Context, j job) (*schedulerpb.Job, error) {
	name := getSchedulerResourceName(c.project, c.region, j.Name)
	uri := triggerUri(c.project, c.region, j.Name)
	res, err := c.cscclient.CreateJob(ctx, &schedulerpb.CreateJobRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", c.project, c.region),
		Job: &schedulerpb.Job{
			Name:            name,
			Description:     fmt.Sprintf("Trigger for %s (created by gruns)", j.Name),
			Target:          targetFromUri(c.defaultTriggerAccount, uri),
			Schedule:        j.Schedule,
			TimeZone:        defaultTimezone,
			UserUpdateTime:  nil,
			State:           schedulerpb.Job_ENABLED,
			Status:          nil,
			ScheduleTime:    nil,
			LastAttemptTime: nil,
			RetryConfig:     nil,
			AttemptDeadline: nil,
		},
	})

	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("scheduler job created: %s", res.Name)

	return res, nil
}

func triggerUri(project, region, name string) string {
	return fmt.Sprintf("https://%s-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/%s/jobs/%s:run", region, project, name)
}

func targetFromUri(triggerServiceAccount, uri string) *schedulerpb.Job_HttpTarget {
	return &schedulerpb.Job_HttpTarget{HttpTarget: &schedulerpb.HttpTarget{
		Uri:        uri,
		HttpMethod: schedulerpb.HttpMethod_POST,
		Headers:    map[string]string{"User-Agent": "Google-Cloud-Scheduler"},
		Body:       nil,
		AuthorizationHeader: &schedulerpb.HttpTarget_OauthToken{
			OauthToken: &schedulerpb.OAuthToken{
				ServiceAccountEmail: triggerServiceAccount,
				Scope:               "https://www.googleapis.com/auth/cloud-platform",
			},
		},
	}}
}
