package argocd

import (
	"errors"
	"sync"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type Client interface {
	SyncApplication(gitopsAppName string) (*argoappv1.Application, error)
	WaitForAppHealthy(appName string, wg *sync.WaitGroup, timeout time.Duration) error
}

type Connection struct {
	Address string
	Token   string
}

type ApiClient struct {
	appClient application.ApplicationServiceClient
	logger    log.FieldLogger
}

func NewClient(c *Connection, logger log.FieldLogger) (*ApiClient, error) {
	if c.Address == "" {
		return &ApiClient{}, errors.New("no argocd address provided")
	}
	if c.Token == "" {
		return &ApiClient{}, errors.New("no argocd token provided")
	}

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

	return &ApiClient{
		appClient: appClient,
		logger:    logger,
	}, nil
}
