package main

import (
	"cloud.google.com/go/run/apiv2/runpb"
	"context"
	"fmt"
	"github.com/elliotchance/pie/v2"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
	"google.golang.org/genproto/googleapis/api"
	"google.golang.org/protobuf/types/known/durationpb"
	"reflect"
	"strings"
	"time"
)

func (c *service) getOrCreateRunJob(ctx context.Context, j job) (*runpb.Job, error) {
	job, err := c.getRunJob(ctx, j.Name)
	if err != nil {
		return c.createRunJob(ctx, j)
	}
	return job, nil
}

func (c *service) getRunJob(ctx context.Context, jobName string) (*runpb.Job, error) {
	name := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", c.project, c.region, jobName)
	return c.jobclient.GetJob(ctx, &runpb.GetJobRequest{Name: name})
}

func (c *service) createRunJob(ctx context.Context, j job) (*runpb.Job, error) {
	res, err := c.jobclient.CreateJob(ctx, &runpb.CreateJobRequest{
		Parent:       fmt.Sprintf("projects/%s/locations/%s", c.project, c.region),
		Job:          createRunJobFromJob(j),
		JobId:        j.Name,
		ValidateOnly: false,
	})

	if err != nil {
		return nil, err
	}

	log.Info().Msgf("cloud run job created: %s", j.Name)

	return res.Wait(ctx)
}

func (c *service) parent() string {
	return fmt.Sprintf("projects/%s/locations/%s", c.project, c.region)
}

func trimParent(parent, s string) string {
	return strings.TrimPrefix(s, parent+"/jobs/")
}

func convertToRunJob(defaultServiceAccount string, j job) job {
	if j.Memory == "" {
		j.Memory = defaultMem
	}

	if j.Cpu == "" {
		j.Cpu = defaultCpu
	}

	if j.ServiceAccount == "" {
		j.ServiceAccount = defaultServiceAccount
	}

	if j.Tasks == 0 {
		j.Tasks = defaultTasks
	}

	if j.Parallelism == 0 {
		j.Parallelism = defaultParallelism
	}

	if j.Timeout == 0 {
		j.Timeout = defaultTimeout
	}

	if j.Retries == 0 {
		j.Retries = defaultRetries
	}

	return j
}

func createRunJobFromJob(j job) *runpb.Job {
	/*var args []string
	if j.Args != "" {
		args = strings.Split(j.Args, " ")
	}*/

	return &runpb.Job{
		//Name:        fmt.Sprintf("%s", j.Name),
		Generation:  0,
		Labels:      map[string]string{"managed_by": tag},
		Annotations: nil,
		LaunchStage: api.LaunchStage_BETA,
		Template: &runpb.ExecutionTemplate{
			Labels:      nil,
			Annotations: nil,
			Parallelism: int32(j.Parallelism),
			TaskCount:   int32(j.Tasks),
			Template: &runpb.TaskTemplate{
				Containers: []*runpb.Container{
					{
						Name:    "",
						Image:   j.Image,
						Command: nil,
						Args:    nil,
						Env:     convertEnvVars(j.Env),
						Resources: &runpb.ResourceRequirements{
							Limits: map[string]string{
								"memory": j.Memory,
								"cpu":    j.Cpu,
							},
							CpuIdle: false,
						},
						Ports:         nil,
						VolumeMounts:  nil,
						WorkingDir:    "",
						LivenessProbe: nil,
						StartupProbe:  nil,
					},
				},
				Volumes: nil,
				Retries: &runpb.TaskTemplate_MaxRetries{MaxRetries: int32(j.Retries)},
				Timeout: &durationpb.Duration{
					Seconds: int64(j.Timeout),
					Nanos:   0,
				},
				ServiceAccount:       j.ServiceAccount,
				ExecutionEnvironment: runpb.ExecutionEnvironment_EXECUTION_ENVIRONMENT_GEN2,
				EncryptionKey:        "",
			},
		},
	}
}

func convertEnvVars(envVars []envVar) []*runpb.EnvVar {
	if len(envVars) == 0 {
		return nil
	}
	var envs []*runpb.EnvVar
	for _, e := range envVars {
		if e.Value != "" && e.Secret != "" {
			log.Fatal().Msgf("env var %s has both value and secret set", e.Name)
		}
		if e.Value != "" {
			envs = append(envs, &runpb.EnvVar{
				Name:   e.Name,
				Values: &runpb.EnvVar_Value{Value: e.Value},
			})
		}

		if e.Secret != "" {
			version := "latest"
			if e.SecretVersion != "" {
				version = e.SecretVersion
			}
			envs = append(envs, &runpb.EnvVar{
				Name:   e.Name,
				Values: &runpb.EnvVar_ValueSource{ValueSource: &runpb.EnvVarSource{SecretKeyRef: &runpb.SecretKeySelector{Secret: e.Secret, Version: version}}},
			})
		}

	}
	return envs
}

func updateJob(runJob *runpb.Job, j job) []string {
	var fieldMask []string
	if runJob.Template.TaskCount != int32(j.Tasks) {
		runJob.Template.TaskCount = int32(j.Tasks)
		fieldMask = append(fieldMask, "template.task_count")
	}

	if runJob.Template.Parallelism != int32(j.Parallelism) {
		runJob.Template.Parallelism = int32(j.Parallelism)
		fieldMask = append(fieldMask, "template.parallelism")
	}

	if runJob.Template.Template.Timeout.GetSeconds() != int64(j.Timeout) {
		runJob.Template.Template.Timeout = durationpb.New(time.Second * time.Duration(j.Timeout))
		fieldMask = append(fieldMask, "template.template.timeout")
	}

	if strings.Join(runJob.Template.Template.Containers[0].Args, " ") != strings.Join(strings.Split(j.Args, " "), " ") {
		runJob.Template.Template.Containers[0].Args = strings.Split(j.Args, " ")
		fieldMask = append(fieldMask, "template.template.containers")
	}

	if runJob.Template.Template.Containers[0].Image != j.Image {
		runJob.Template.Template.Containers[0].Image = j.Image
		fieldMask = append(fieldMask, "template.template.containers.0.image")
	}

	if runJob.Template.Template.Containers[0].Resources.Limits["memory"] != j.Memory {
		runJob.Template.Template.Containers[0].Resources.Limits["memory"] = j.Memory
		fieldMask = append(fieldMask, "template.template.containers.0.resources.limits.memory")
	}

	if runJob.Template.Template.Containers[0].Resources.Limits["cpu"] != j.Cpu {
		runJob.Template.Template.Containers[0].Resources.Limits["cpu"] = j.Cpu
		fieldMask = append(fieldMask, "template.template.containers.0.resources.limits.cpu")
	}

	if !reflect.DeepEqual(runJob.Template.Template.Containers[0].Env, convertEnvVars(j.Env)) {
		runJob.Template.Template.Containers[0].Env = convertEnvVars(j.Env)
		fieldMask = append(fieldMask, "template.template.containers.0.env")
	}

	var args []string
	if j.Args != "" {
		args = strings.Split(j.Args, " ")
	}

	if !compareArgs(runJob.Template.Template.Containers[0].Args, args) {
		runJob.Template.Template.Containers[0].Args = args
		fieldMask = append(fieldMask, "template.template.containers.0.args")
	}

	if runJob.Template.Template.ServiceAccount != j.ServiceAccount {
		runJob.Template.Template.ServiceAccount = j.ServiceAccount
		fieldMask = append(fieldMask, "template.template.service_account")
	}

	if runJob.Template.Template.Retries.(*runpb.TaskTemplate_MaxRetries).MaxRetries != int32(j.Retries) {
		runJob.Template.Template.Retries = &runpb.TaskTemplate_MaxRetries{MaxRetries: int32(j.Retries)}
		fieldMask = append(fieldMask, "template.template.retries")
	}

	return fieldMask
}

func compareArgs(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func handleRunJob(c *service, j job) error {
	ctx := context.Background()
	rjob, err := c.getOrCreateRunJob(ctx, j)
	if err != nil {
		return err
	}

	fieldMask := updateJob(rjob, j)
	if len(fieldMask) > 0 {
		log.Info().Msgf("updating job %s with fieldmask %s", j.Name, fieldMask)
		_, err := c.jobclient.UpdateJob(ctx, &runpb.UpdateJobRequest{Job: rjob})
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteRunJobs(c *service, validJobNames []string) error {
	ctx := context.Background()
	iterJobs := c.jobclient.ListJobs(ctx, &runpb.ListJobsRequest{
		Parent:    c.parent(),
		PageSize:  500,
		PageToken: "",
	})

	for {
		res, err := iterJobs.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return err
		}

		if !pie.Contains(validJobNames, trimParent(c.parent(), res.Name)) && res.Labels["managed_by"] == tag {
			log.Info().Msgf("deleting job %s ", res.Name)
			ops, err := c.jobclient.DeleteJob(ctx, &runpb.DeleteJobRequest{Name: res.Name})
			if err != nil {
				return err
			}
			res, err = ops.Wait(ctx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
