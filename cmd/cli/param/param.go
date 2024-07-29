package param

type GitOpts struct {
	Branch string `arg:"-b,--branch,env:DEFAULT_BRANCH"`
	Sha    string `arg:"-s,--sha,env:DEFAULT_SHA"`
}

type FunctionArg struct {
	Path string `arg:"positional" help:"path to function" default:"."`
}

type Init struct {
	Language string `arg:"positional" help:"Language to scaffold"`
	Name     string `arg:"positional" help:"Name of function"`
}

type Build struct {
	SSHAgent bool   `arg:"-a,--ssh-agent" help:"mount ssh-agent into build (not yet implemented)"`
	Context  string `arg:"-c,--build-context" help:"set builtime path, defaults to arg path."`
	Run      bool   `arg:"--run" help:"run the function locally after building"`
	FunctionArg
}

type Publish struct {
	Login            bool `arg:"-l,--ecr-login" help:"Login to ECR"`
	EnsureRepository bool `arg:"--ensure-repository" help:"Ensure ECR repository exists"`
	Build
}

type Deploy struct {
	Enable  bool `arg:"--enable" help:"enable event bus invocation"`
	Disable bool `arg:"--disable" help:"disable event bus invocation"`
	FunctionArg
}

type Destroy struct {
	FunctionArg
}

type Releases struct {
	FunctionArg
}

type Deployments struct {
	FunctionArg
}

type DeployTime struct {
	FunctionArg
}

type BuildTime struct {
	FunctionArg
}

type Inspect struct {
	Build  *BuildTime  `arg:"subcommand:build" help:"View encoded buildtime config"`
	Deploy *DeployTime `arg:"subcommand:deploy" help:"View parsed deploytime config"`
}
