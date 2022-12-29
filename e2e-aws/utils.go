// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	testIdentifier     = "e2e-aws"
	serverPort         = 9999
	webhookHandlerPath = "/webhook"
	provisionerBaseURL = "http://127.0.0.1:8075"
)

var webhookServerURL = fmt.Sprintf("http://127.0.0.1:%d%s", serverPort, webhookHandlerPath)

func formatURL(baseURL string, path ...string) string {
	result, _ := url.JoinPath(baseURL, path...)
	return result
}

func checkProvisionerRunning(url string) error {
	res, err := http.Get(formatURL(provisionerBaseURL, "/api/clusters"))
	if err != nil {
		return errors.Wrap(err, "error contacting provisioner")
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("error contacting provisioner")
	}

	return nil
}

func checkDependencies(t *testing.T) {
	log.Println("Checking if provisioner is running")
	if err := checkProvisionerRunning(provisionerBaseURL); err != nil {
		log.Println("provisioner seems offline!")
		t.Error(err.Error())
		t.FailNow()
	}
}

func createUniqueName() string {
	return fmt.Sprintf("e2e-aws-%s", uuid.New().String())
}

func SetupTest(t *testing.T) *ClusterTestSuite {
	suite := NewClusterTestSuite()

	checkDependencies(t)

	if err := suite.StartServer(context.TODO()); err != nil {
		log.Println("failed to start http server for webhook events")
		log.Println(err)
		t.FailNow()
	}

	if err := suite.RegisterWebhook(); err != nil {
		log.Println("error registering the webhook, I'm going to assume it is already created")
	}

	return suite
}

func CleanupTest(t *testing.T, suite *ClusterTestSuite) {
	if err := suite.UnregisterWebhook(); err != nil {
		log.Println(err.Error())
	}

	suite.StopServer()

	// // Removing dangling clusters
	// clusters, err := suite.Client().GetClusters(&model.GetClustersRequest{})
	// if err != nil {
	// 	log.Println(err.Error())
	// 	return
	// }

	// for _, cluster := range clusters {
	// 	if err := suite.Client().DeleteCluster(cluster.ID); err != nil {
	// 		log.Printf("error removing cluster: %s", err.Error())
	// 	}
	// }
}
