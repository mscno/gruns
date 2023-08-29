package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
	"time"
)

const (
	tag = "gruns-cli"
)

const (
	defaultTimezone           = "UTC"
	defaultJobDefinitionsFile = "jobs.yml"
	defaultMem                = "512Mi"
	defaultCpu                = "1000m"
	defaultTasks              = 1
	defaultParallelism        = 1
	defaultTimeout            = 900
	defaultRetries            = 1
)

func main() {
	var region string
	var projectName string
	var projectNumber string
	var disableTriggers bool
	var fileName string
	var serviceAccount string

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	logger := zerolog.New(consoleWriter).With().Timestamp().Logger()
	log.Logger = logger

	app := &cli.App{
		Name: "gruns",
		Commands: []*cli.Command{
			{
				Name: "apply",
				Action: func(cCtx *cli.Context) error {
					fileName = defaultJobDefinitionsFile
					if cCtx.NArg() > 0 {
						fileName = cCtx.Args().Get(0)
					}

					return apply(args{
						ProjectId:       projectName,
						ProjectNumber:   projectNumber,
						Region:          region,
						DisableTriggers: disableTriggers,
						FileName:        fileName,
						ServiceAccount:  serviceAccount,
					})
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "project-id",
						Usage:       "GCP Project ID",
						Destination: &projectName,
						EnvVars:     []string{"GOOGLE_PROJECT_ID"},
					},
					&cli.StringFlag{
						Name:        "project-number",
						Usage:       "GCP Project Number",
						Destination: &projectNumber,
						EnvVars:     []string{"GOOGLE_PROJECT_NUMBER"},
					},
					&cli.StringFlag{
						Name:        "region",
						Usage:       "GCP Region",
						Destination: &region,
						EnvVars:     []string{"GOOGLE_REGION"},
					},
					&cli.StringFlag{
						Name:        "service-account",
						Usage:       "Service Account Email",
						Destination: &region,
						EnvVars:     []string{"GOOGLE_SERVICE_ACCOUNT"},
					},
					&cli.BoolFlag{
						Name:        "disable-triggers",
						Usage:       "Flag to disable trigger activation",
						Destination: &disableTriggers,
					},
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}

}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatal().Msgf("%s env var is not set", key)
	}
	return val
}

func apply(args args) error {
	if args.ServiceAccount == "" {
		args.ServiceAccount = fmt.Sprintf("%s-compute@developer.gserviceaccount.com", args.ProjectNumber)
	}
	if args.TriggerAccount == "" {
		args.TriggerAccount = fmt.Sprintf("%s-compute@developer.gserviceaccount.com", args.ProjectNumber)
	}

	fmt.Println("serviceAccount: ", args.ServiceAccount)
	fmt.Println("triggerServiceAccount: ", args.TriggerAccount)
	fmt.Println("projectId: ", args.ProjectId)
	fmt.Println("projectNumber: ", args.ProjectNumber)

	var jobNames []string
	var triggerNames []string

	ctx := context.Background()
	svc := initializeService(ctx, args)
	jobs, err := readJobs(args.FileName)
	if err != nil {
		return err
	}

	jobs = interpolateJobs(args, jobs)

	for _, j := range jobs {
		j = convertToRunJob(args.ServiceAccount, j)
		if err != nil {
			return errors.Wrapf(err, "invalid job: %s", j.Name)
		}

		if j.Schedule != "" {
			triggerNames = append(triggerNames, j.Name+"-trigger")
		}
		jobNames = append(jobNames, j.Name)

		// Get and Create/update scheduler job
		if j.Schedule != "" {
			err = handleSchedulerJob(svc, j)
			if err != nil {
				return errors.Wrapf(err, "scheduler job error: %s", j.Name)
			}
		}

		// Get and Create/update run job
		err = handleRunJob(svc, j)
		if err != nil {
			return errors.Wrapf(err, "run job error: %s", j.Name)
		}
	}

	return svc.cleanup(triggerNames, jobNames)
}

func (svc *service) cleanup(triggerNames, jobNames []string) error {
	// Delete all scheduler jobs that are not defined in yaml (only jobs managed by jobs cli)
	err := deleteSchedulerJobs(svc, triggerNames)
	if err != nil {
		return errors.Wrapf(err, "delete scheduler jobs error")
	}
	// Delete all run jobs that are not defined in yaml (only jobs managed by jobs cli)
	err = deleteRunJobs(svc, jobNames)
	if err != nil {
		return errors.Wrapf(err, "delete run jobs error")
	}
	return nil
}
