package cli

import (
	"context"

	"github.com/linecard/self/convention/config"
	"github.com/linecard/self/sdk"

	"github.com/alexflint/go-arg"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/jedib0t/go-pretty/table"
)

var cfg config.Config
var api sdk.API
var stsc *sts.Client
var tablec table.Writer

type NullCommand struct{}

type DeploymentOpts struct {
	// Default: the current branch name.
	Tag string `arg:"-t,--tag,env:DEFAULT_DEPLOYMENT_TAG"`
	// Default: the current branch name.
	NameSpace string `arg:"-n,--namespace,env:DEFAULT_DEPLOYMENT_NAMESPACE"`
}

type ReleaseOpts struct {
	// Default: the current branch name.
	Branch string `arg:"-b,--branch,env:DEFAULT_RELEASE_BRANCH"`
	// Default: the current commit sha.
	Sha string `arg:"-s,--sha,env:DEFAULT_RELEASE_SHA"`
}

func Invoke(ctx context.Context) {
	BeforeAll(ctx)

	if cfg.Function != nil {
		var f FunctionScope
		arg.MustParse(&f)
		f.Handle(ctx)
		return
	}

	var r RepoScope
	arg.MustParse(&r)
	r.Handle(ctx)
}
