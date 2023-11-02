package argocd

import (
	"encoding/base64"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type ArgoClusterRegisterTestSuite struct {
	suite.Suite
	argoK8sFile        *Argock8sRegister
	updatedArgok8sFile *Argock8sRegister
	newArgoK8sFile     *ArgocdClusterRegisterParameters
	tempDir            string
	filePath           string
}

func (suite *ArgoClusterRegisterTestSuite) SetupSuite() {
	var err error
	suite.tempDir, err = os.MkdirTemp("", "cluster-register-")
	if err != nil {
		assert.Errorf(suite.T(), err, "failed to create temporary directory")
	}
	suite.filePath = path.Join(suite.tempDir, "cluster-values.yaml")

	newClusterLabels := ArgocdClusterLabels{
		ClusterTypes: "customer",
		ClusterID:    "1234567",
	}

	suite.argoK8sFile = &Argock8sRegister{
		Clusters: []ArgocdClusterRegisterParameters{
			{
				Name:      "cluster1",
				Type:      "kops",
				Labels:    newClusterLabels,
				APIServer: "cluster1.test.com",
				CaData:    base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCTUCgR5+EsitNh-----END PRIVATE KEY-----")),
				CertData:  base64.StdEncoding.EncodeToString([]byte("-----BEGIN CERTIFICATE-----MIIDGTCCAgGgAwIBAgIUHqQQpkxCJ/xg6G/PVyFFEYrBPjswDQYJKoZIhvcNAQEL-----END CERTIFICATE-----")),
				KeyData:   base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDNPAqO0X1O7gw-----END PRIVATE KEY-----")),
			},
		},
	}
	suite.newArgoK8sFile = &ArgocdClusterRegisterParameters{
		Name:      "cluster2",
		Type:      "kops",
		Labels:    newClusterLabels,
		APIServer: "cluster2.test.com",
		CaData:    base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCTUCgR5+EsitNh-----END PRIVATE KEY-----")),
		CertData:  base64.StdEncoding.EncodeToString([]byte("-----BEGIN CERTIFICATE-----MIIDGTCCAgGgAwIBAgIUHqQQpkxCJ/xg6G/PVyFFEYrBPjswDQYJKoZIhvcNAQEL-----END CERTIFICATE-----")),
		KeyData:   base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDNPAqO0X1O7gw-----END PRIVATE KEY-----")),
	}

	suite.updatedArgok8sFile = &Argock8sRegister{
		Clusters: []ArgocdClusterRegisterParameters{
			{
				Name:      "cluster1",
				Type:      "kops",
				Labels:    newClusterLabels,
				APIServer: "cluster1.test.com",
				CaData:    base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCTUCgR5+EsitNh-----END PRIVATE KEY-----")),
				CertData:  base64.StdEncoding.EncodeToString([]byte("-----BEGIN CERTIFICATE-----MIIDGTCCAgGgAwIBAgIUHqQQpkxCJ/xg6G/PVyFFEYrBPjswDQYJKoZIhvcNAQEL-----END CERTIFICATE-----")),
				KeyData:   base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDNPAqO0X1O7gw-----END PRIVATE KEY-----")),
			},
			{
				Name:      "cluster2",
				Type:      "kops",
				Labels:    newClusterLabels,
				APIServer: "cluster2.test.com",
				CaData:    base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCTUCgR5+EsitNh-----END PRIVATE KEY-----")),
				CertData:  base64.StdEncoding.EncodeToString([]byte("-----BEGIN CERTIFICATE-----MIIDGTCCAgGgAwIBAgIUHqQQpkxCJ/xg6G/PVyFFEYrBPjswDQYJKoZIhvcNAQEL-----END CERTIFICATE-----")),
				KeyData:   base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDNPAqO0X1O7gw-----END PRIVATE KEY-----")),
			},
		},
	}

	updatedYAML, err := yaml.Marshal(&suite.argoK8sFile)
	if err != nil {
		assert.Errorf(suite.T(), err, "Error marshalling YAML")
	}

	err = os.WriteFile(suite.filePath, updatedYAML, 0644)
	if err != nil {
		assert.Errorf(suite.T(), err, "failed to write cluster file")
	}
}

func (suite *ArgoClusterRegisterTestSuite) TearDownSuite() {
	os.RemoveAll(suite.tempDir)
}

func (suite *ArgoClusterRegisterTestSuite) TestReadArgoK8sRegistrationFile() {
	clusteFile, err := os.ReadFile(suite.filePath)
	if err != nil {
		assert.Errorf(suite.T(), err, "failed to read cluster file")
	}
	var argo Argock8sRegister
	readFile, err := argo.ReadArgoK8sRegistrationFile(clusteFile)
	if err != nil {
		assert.Errorf(suite.T(), err, "Error reading Cluster file")
	}

	assert.Equal(suite.T(), readFile, suite.argoK8sFile)
}

func (suite *ArgoClusterRegisterTestSuite) TestUpdateK8sClusterRegistrationFile() {
	var argo Argock8sRegister
	argo.UpdateK8sClusterRegistrationFile(suite.argoK8sFile, *suite.newArgoK8sFile, suite.filePath)

	clusteFile, err := os.ReadFile(suite.filePath)
	if err != nil {
		assert.Errorf(suite.T(), err, "failed to read cluster file")
	}
	readFile, err := argo.ReadArgoK8sRegistrationFile(clusteFile)
	if err != nil {
		assert.Errorf(suite.T(), err, "Error reading Cluster file")
	}

	assert.Equal(suite.T(), readFile, suite.updatedArgok8sFile)
}

func TestArgoClusterRegisterTestSuite(t *testing.T) {
	suite.Run(t, new(ArgoClusterRegisterTestSuite))

}
