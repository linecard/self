package param

type FunctionArg struct {
	Name string `arg:"positional" help:"Name of function"`
}

type GitOpts struct {
	Branch string `arg:"-b,--branch,env:DEFAULT_DEPLOYMENT_TAG"`
	Sha    string `arg:"-s,--sha,env:DEFAULT_DEPLOYMENT_NAMESPACE"`
}

type Init struct {
	Language string `arg:"positional" help:"Language to scaffold"`
	FunctionArg
}

type Build struct {
	SSHAgent bool   `arg:"-a,--ssh-agent" help:"mount ssh-agent into build (TODO)"`
	Context  string `arg:"-c,--context" help:"build context path" default:"."`
	GitOpts
	FunctionArg
}

type Release struct {
	Login            bool `arg:"-l,--ecr-login" help:"Login to ECR"`
	EnsureRepository bool `arg:"-r,--ensure-repository,env:DEFAULT_ENSURE_REPOSITORY"`
	Build
	GitOpts
	FunctionArg
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
}

type Deployments struct {
	GitOpts
}

type Config struct {
	GitOpts
}
