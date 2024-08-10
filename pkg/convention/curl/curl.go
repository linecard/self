package curl

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/linecard/self/pkg/convention/config"
)

type Sigv4Service interface {
	SignRequest(ctx context.Context, request *http.Request) (err error)
	MockIAMAuthRequestCtx(ctx context.Context, callerArn string, req *http.Request) (err error)
}

type Service struct {
	Sigv4 Sigv4Service
}

type Convention struct {
	Config  config.Config
	Service Service
}

func FromServices(c config.Config, sigv4 Sigv4Service) Convention {
	return Convention{
		Config: c,
		Service: Service{
			Sigv4: sigv4,
		},
	}
}

func (c Convention) Signed(ctx context.Context, method string, url string, data []byte) (response *http.Response, err error) {
	request, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return
	}

	if strings.Contains(request.URL.Host, "localhost") || strings.Contains(request.URL.Host, "127.0.0.1") {
		if err = c.Service.Sigv4.MockIAMAuthRequestCtx(ctx, c.Config.Caller.Arn, request); err != nil {
			return
		}
	} else {
		if err = c.Service.Sigv4.SignRequest(ctx, request); err != nil {
			return
		}
	}

	return http.DefaultClient.Do(request)
}
