package main

type root struct {
	Jobs []job
}

type job struct {
	Name           string
	ServiceAccount string `yaml:"service_account"`
	Parallelism    int
	Tasks          int
	Retries        int
	Timeout        int
	Image          string
	Schedule       string
	Args           string
	Cpu            string
	Memory         string
	Env            []envVar
}

type envVar struct {
	Name          string
	Value         string
	Secret        string
	SecretVersion string `yaml:"secret_version"`
}

type args struct {
	ProjectId       string
	ProjectNumber   string
	Region          string
	DisableTriggers bool
	ServiceAccount  string
	TriggerAccount  string
	FileName        string
}
