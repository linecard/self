package param

type FunctionArg struct {
	Path string `arg:"positional" help:"path to function" default:"."`
}

type GitOpts struct {
	Branch string `arg:"-b,--branch,env:DEFAULT_BRANCH"`
	Sha    string `arg:"-s,--sha,env:DEFAULT_SHA"`
}

type Init struct {
	Language string `arg:"positional" help:"Language to scaffold"`
	Name     string `arg:"positional" help:"Name of function"`
}

type Build struct {
	SSHAgent bool   `arg:"-a,--ssh-agent" help:"mount ssh-agent into build (TODO)"`
	Context  string `arg:"-c,--build-context,env:DEFAULT_BUILD_CONTEXT" help:"build context path"`
	GitOpts
	FunctionArg
}

type Publish struct {
	Login            bool `arg:"-l,--ecr-login" help:"Login to ECR"`
	EnsureRepository bool `arg:"-r,--ensure-repository,env:DEFAULT_ENSURE_REPOSITORY"`
	Build
	GitOpts
}

type Deploy struct {
	GitOpts
	FunctionArg
}

type Destroy struct {
	GitOpts
	FunctionArg
}

type Releases struct {
	GitOpts
	FunctionArg
}

type Deployments struct {
	GitOpts
	FunctionArg
}

type DeployTime struct {
	GitOpts
	FunctionArg
}

type BuildTime struct {
	Global bool `arg:"-g,--global" help:"show discovered config"`
	FunctionArg
}

type Inspect struct {
	Build  *BuildTime  `arg:"subcommand:build" help:"View encoded buildtime config"`
	Deploy *DeployTime `arg:"subcommand:deploy" help:"View parsed deploytime config"`
}
