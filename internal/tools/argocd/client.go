package argocd

import (
	"sync"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type Client interface {
	SyncApplication(gitopsAppName, appName string) (*argoappv1.Application, error)
	WaitForAppHealthy(appName string, wg *sync.WaitGroup, timeout time.Duration)
}

type Connection struct {
	Address string
	Token   string
}

type AppClient struct {
	appClient application.ApplicationServiceClient
	logger    log.FieldLogger
}

func NewClient(c *Connection, logger log.FieldLogger) (*AppClient, error) {
	apiClient, err := apiclient.NewClient(&apiclient.ClientOptions{
		ServerAddr: c.Address,
		Insecure:   true,
		AuthToken:  c.Token,
		GRPCWeb:    true,
	})
	if err != nil {
		return nil, err
	}

	_, appClient, err := apiClient.NewApplicationClient()
	if err != nil {
		return nil, err
	}

	return &AppClient{
		appClient: appClient,
		logger:    logger,
	}, nil
}
