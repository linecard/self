package param

type GlobalOpts struct {
	Branch                 string `arg:"--branch,env:SELF_BRANCH_OVERRIDE"`
	Sha                    string `arg:"--sha,env:SELF_SHA_OVERRIDE"`
	EcrId                  string `arg:"--ecr-id,env:SELF_ECR_REGISTRY_ID"`
	EcrRegion              string `arg:"--ecr-region,env:SELF_ECR_REGISTRY_REGION"`
	ApiGatewayId           string `arg:"--api-gateway-id,env:SELF_API_GATEWAY_ID"`
	ApiGatewayAuthType     string `arg:"--api-gateway-auth-type,env:SELF_API_GATEWAY_AUTH_TYPE"`
	ApiGatewayAuthorizerId string `arg:"--api-gateway-authorizer-id,env:SELF_API_GATEWAY_AUTHORIZER_ID"`
	SelfBusName            string `arg:"--bus-name,env:SELF_SELF_BUS_NAME"`
	SubnetIds              string `arg:"--subnet-ids,env:SELF_SUBNET_IDS"`
	SecurityGroupIds       string `arg:"--security-group-ids,env:SELF_SECURITY_GROUP_IDS"`
	OwnerPrefixResources   bool   `arg:"--prefix-resources-with-owner,env:SELF_PREFIX_RESOURCES_WITH_OWNER"`
	OwnerPrefixRoutes      bool   `arg:"--prefix-routes-with-owner,env:SELF_PREFIX_ROUTE_KEY_WITH_OWNER"`
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
	Login            bool     `arg:"-l,--ecr-login" help:"Login to ECR"`
	EnsureRepository bool     `arg:"--ensure-repository" help:"Ensure ECR repository exists"`
	Force            bool     `arg:"-f,--force" help:"Override dirty commit protection"`
	EmitDeploy       bool     `arg:"--emit-deploy,env:SELF_EMIT_DEPLOY_ON_PUBLISH" help:"Emit deploy event"`
	ExceptAccounts   []string `arg:"--except" help:"Exclude deployment to these accounts when emitting deploy event"`
	Build
}

type Deploy struct {
	Enable  bool `arg:"--enable,env:SELF_ENABLE_ON_DEPLOY" help:"enable event bus invocation"`
	Disable bool `arg:"--disable,env:SELF_DISABLE_ON_DEPLOY" help:"disable event bus invocation"`
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
	EmitDestroy bool `arg:"--emit-destroy,env:SELF_EMIT_DESTROY_ON_UNTAG" help:"Emit destroy event"`
}
