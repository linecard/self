package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/umwelt"
)

func FromCwd(ctx context.Context, cwd string, gitMock gitlib.DotGit, awsConfig aws.Config) umwelt.Here {
	return umwelt.Here{
		Caller: umwelt.ThisCaller{
			Id:      "user-123",
			Arn:     "arn:aws:iam::123456789012:user/test",
			Account: "123456789012",
			Region:  "us-west-2",
		},
		Git: gitMock,
		Registry: umwelt.ThisRegistry{
			Id:     "123456789013",
			Region: "us-west-2",
		},
		ApiGateway: umwelt.ThisApiGateway{
			Id: "mockApiId",
		},
		Function:  umwelt.Selfish(cwd),
		Functions: umwelt.SelfDiscovery(gitMock.Root),
	}
}
