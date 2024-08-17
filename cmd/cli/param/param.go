package param

type GlobalOpts struct {
	Branch               string `arg:"--branch,env:GIT_BRANCH_OVERRIDE"`
	Sha                  string `arg:"--sha,env:GIT_SHA_OVERRIDE"`
	EcrId                string `arg:"--ecr-id,env:AWS_ECR_REGISTRY_ID"`
	EcrRegion            string `arg:"--ecr-region,env:AWS_ECR_REGISTRY_REGION"`
	ApiGatewayId         string `arg:"--api-gateway-id,env:AWS_API_GATEWAY_ID"`
	SubnetIds            string `arg:"--subnet-ids,env:AWS_SUBNET_IDS"`
	SecurityGroupIds     string `arg:"--security-group-ids,env:AWS_SECURITY_GROUP_IDS"`
	OwnerPrefixResources bool   `arg:"--prefix-resources-with-owner,env:AWS_PREFIX_RESOURCES_WITH_OWNER"`
	OwnerPrefixRoutes    bool   `arg:"--prefix-routes-with-owner,env:AWS_PREFIX_ROUTE_KEY_WITH_OWNER"`
}

type FunctionArg struct {
	Path string `arg:"positional" help:"path to function" default:"."`
}

type Init struct {
	Scaffold string `arg:"positional,required" help:"go, python, node, ruby or self"`
	Name     string `arg:"positional,required" help:"Release name"`
}

type Build struct {
	SSHAgent bool   `arg:"-a,--ssh-agent" help:"mount ssh-agent into build (not yet implemented)"`
	Context  string `arg:"-c,--context" help:"set builtime path, defaults to arg path."`
	Run      bool   `arg:"--run" help:"run the function locally after building"`
	FunctionArg
}

type Publish struct {
	Login            bool `arg:"-l,--ecr-login" help:"Login to ECR"`
	EnsureRepository bool `arg:"--ensure-repository" help:"Ensure ECR repository exists"`
	Force            bool `arg:"-f,--force" help:"Override dirty commit protection"`
	EmitDeploy       bool `arg:"--emit-deploy" help:"Emit deploy event"`
	Build
}

type Deploy struct {
	Enable  bool `arg:"--enable,env:ENABLE_EVENTING_ON_DEPLOY" help:"enable event bus invocation"`
	Disable bool `arg:"--disable,env:DISABLE_EVENTING_ON_DEPLOY" help:"disable event bus invocation"`
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

type GlobalConfig struct{}

type Inspect struct {
	Build  *BuildTime    `arg:"subcommand:build" help:"print buildtime config for given function"`
	Deploy *DeployTime   `arg:"subcommand:deploy" help:"print deploytime config for given function"`
	Global *GlobalConfig `arg:"subcommand:global" help:"print global config for repository"`
}

type Untag struct {
	FunctionArg
	EmitDestroy bool `arg:"--emit-destroy" help:"Emit destroy event"`
}
