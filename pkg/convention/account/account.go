package account

import (
	"context"

	"github.com/linecard/self/pkg/convention/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type RegistryService interface {
	Token(ctx context.Context, registryId string) (string, error)
}

type BuildService interface {
	Login(ctx context.Context, registryUrl, username, password string) error
}

type Services struct {
	Registry RegistryService
	Build    BuildService
}

type Convention struct {
	Config  config.Config
	Service Services
}

func FromServices(c config.Config, b BuildService, r RegistryService) Convention {
	return Convention{
		Config: c,
		Service: Services{
			Registry: r,
			Build:    b,
		},
	}
}

func (c Convention) LoginToEcr(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "ecr-login")
	defer span.End()

	span.SetAttributes(
		attribute.String("registry-url", c.Config.Registry.Url),
		attribute.String("registry-id", c.Config.Registry.Id),
	)

	token, err := c.Service.Registry.Token(ctx, c.Config.Registry.Id)
	if err != nil {
		return err
	}

	if err := c.Service.Build.Login(ctx, c.Config.Registry.Url, "AWS", token); err != nil {
		return err
	}

	return nil
}
