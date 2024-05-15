package account

import (
	"context"

	"github.com/linecard/self/convention/config"
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
	token, err := c.Service.Registry.Token(ctx, c.Config.Registry.Id)
	if err != nil {
		return err
	}

	if err := c.Service.Build.Login(ctx, c.Config.Registry.Url, "AWS", token); err != nil {
		return err
	}

	return nil
}
