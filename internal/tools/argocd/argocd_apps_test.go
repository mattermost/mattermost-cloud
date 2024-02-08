package argocd

import (
	"os"
	"reflect"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"gopkg.in/yaml.v3"
)

func TestAddClusterIDLabel(t *testing.T) {
	tmpfile, err := os.CreateTemp(".", "testyaml")
	if err != nil {
		t.Fatalf("Error creating temporary YAML file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	testYAML := []byte(`
applications:
  - name: tempo
    namespace: tempo
    repo: https://grafana.github.io/helm-charts
    path: tempo-distributed
    revision: 1.0.1
    helm:
      enabled: true
      additionalManifests: false
    cluster_labels:
      - cluster-type: cnc
  - name: loki
    namespace: loki
    repo: https://grafana.github.io/helm-charts
    path: loki-distributed
    revision: 0.69.16
    helm:
      enabled: true
      additionalManifests: true
    cluster_labels:
      - cluster-type: cnc
`)
	if _, err = tmpfile.Write(testYAML); err != nil {
		t.Fatalf("Error writing to temporary YAML file: %v", err)
	}

	yamlFile, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Error reading temporary YAML file: %v", err)
	}

	var yamlData map[string][]Application
	err = yaml.Unmarshal(yamlFile, &yamlData)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML file: %v", err)
	}

	logger := testlib.MakeLogger(t)
	AddClusterIDLabel(yamlData, "loki", "cluster1", logger)

	expectedModifiedYAML := `
applications:
  - name: tempo
    namespace: tempo
    repo: https://grafana.github.io/helm-charts
    path: tempo-distributed
    revision: 1.0.1
    helm:
      enabled: true
      additionalManifests: false
    cluster_labels:
      - cluster-type: cnc
  - name: loki
    namespace: loki
    repo: https://grafana.github.io/helm-charts
    path: loki-distributed
    revision: 0.69.16
    helm:
      enabled: true
      additionalManifests: true
    cluster_labels:
      - cluster-type: cnc
      - cluster-id: cluster1
`
	modifiedYAML, err := yaml.Marshal(&yamlData)
	if err != nil {
		t.Fatalf("Error marshalling YAML: %v", err)
	}

	var expectedData map[string]interface{}
	var modifiedData map[string]interface{}

	if err := yaml.Unmarshal([]byte(expectedModifiedYAML), &expectedData); err != nil {
		t.Fatalf("Error unmarshaling expected modified YAML: %v", err)
	}

	if err := yaml.Unmarshal(modifiedYAML, &modifiedData); err != nil {
		t.Fatalf("Error unmarshaling modified YAML: %v", err)
	}

	if !reflect.DeepEqual(modifiedData, expectedData) {
		t.Fatalf("Expected modified YAML: %v, got: %v", expectedData, modifiedData)
	}
}

func TestRemoveClusterIDLabel(t *testing.T) {
	tmpfile, err := os.CreateTemp(".", "testyaml")
	if err != nil {
		t.Fatalf("Error creating temporary YAML file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	testYAML := []byte(`
applications:
  - name: tempo
    namespace: tempo
    repo: https://grafana.github.io/helm-charts
    path: tempo-distributed
    revision: 1.0.1
    helm:
      enabled: true
      additionalManifests: false
    cluster_labels:
      - cluster-type: cnc
  - name: loki
    namespace: loki
    repo: https://grafana.github.io/helm-charts
    path: loki-distributed
    revision: 0.69.16
    helm:
      enabled: true
      additionalManifests: true
    cluster_labels:
      - cluster-type: cnc
      - cluster-id: cluster1
`)
	if _, err = tmpfile.Write(testYAML); err != nil {
		t.Fatalf("Error writing to temporary YAML file: %v", err)
	}

	yamlFile, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Error reading temporary YAML file: %v", err)
	}

	var data map[string][]Application
	err = yaml.Unmarshal(yamlFile, &data)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML file: %v", err)
	}

	logger := testlib.MakeLogger(t)
	RemoveClusterIDLabel(data, "loki", "cluster1", logger)

	expectedModifiedYAML := []byte(`
applications:
  - name: tempo
    namespace: tempo
    repo: https://grafana.github.io/helm-charts
    path: tempo-distributed
    revision: 1.0.1
    helm:
      enabled: true
      additionalManifests: false
    cluster_labels:
      - cluster-type: cnc
  - name: loki
    namespace: loki
    repo: https://grafana.github.io/helm-charts
    path: loki-distributed
    revision: 0.69.16
    helm:
      enabled: true
      additionalManifests: true
    cluster_labels:
      - cluster-type: cnc
`)
	modifiedYAML, err := yaml.Marshal(&data)
	if err != nil {
		t.Fatalf("Error marshalling YAML: %v", err)
	}

	var expectedData map[string]interface{}
	var modifiedData map[string]interface{}

	if err := yaml.Unmarshal(expectedModifiedYAML, &expectedData); err != nil {
		t.Fatalf("Error unmarshaling expected modified YAML: %v", err)
	}

	if err := yaml.Unmarshal(modifiedYAML, &modifiedData); err != nil {
		t.Fatalf("Error unmarshaling modified YAML: %v", err)
	}

	if !reflect.DeepEqual(modifiedData, expectedData) {
		t.Fatalf("Expected modified YAML: %v, got: %v", expectedData, modifiedData)
	}
}

func TestClusterIDLabelAlreadyExists(t *testing.T) {
	tmpfile, err := os.CreateTemp(".", "testyaml")
	if err != nil {
		t.Fatalf("Error creating temporary YAML file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	testYAML := []byte(`
applications:
  - name: tempo
    namespace: tempo
    repo: https://grafana.github.io/helm-charts
    path: tempo-distributed
    revision: 1.0.1
    helm:
      enabled: true
      additionalManifests: false
    cluster_labels:
      - cluster-type: cnc
  - name: loki
    namespace: loki
    repo: https://grafana.github.io/helm-charts
    path: loki-distributed
    revision: 0.69.16
    helm:
      enabled: true
      additionalManifests: true
    cluster_labels:
      - cluster-type: cnc
      - cluster-id: cluster1
`)
	if _, err = tmpfile.Write(testYAML); err != nil {
		t.Fatalf("Error writing to temporary YAML file: %v", err)
	}

	yamlFile, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Error reading temporary YAML file: %v", err)
	}

	var data map[string][]Application
	err = yaml.Unmarshal(yamlFile, &data)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML file: %v", err)
	}

	logger := testlib.MakeLogger(t)
	AddClusterIDLabel(data, "loki", "cluster1", logger)

	expectedModifiedYAML := []byte(`
applications:
  - name: tempo
    namespace: tempo
    repo: https://grafana.github.io/helm-charts
    path: tempo-distributed
    revision: 1.0.1
    helm:
      enabled: true
      additionalManifests: false
    cluster_labels:
      - cluster-type: cnc
  - name: loki
    namespace: loki
    repo: https://grafana.github.io/helm-charts
    path: loki-distributed
    revision: 0.69.16
    helm:
      enabled: true
      additionalManifests: true
    cluster_labels:
      - cluster-type: cnc
      - cluster-id: cluster1
`)
	modifiedYAML, err := yaml.Marshal(&data)
	if err != nil {
		t.Fatalf("Error marshalling YAML: %v", err)
	}

	var expectedData map[string]interface{}
	var modifiedData map[string]interface{}

	if err := yaml.Unmarshal(expectedModifiedYAML, &expectedData); err != nil {
		t.Fatalf("Error unmarshaling expected modified YAML: %v", err)
	}

	if err := yaml.Unmarshal(modifiedYAML, &modifiedData); err != nil {
		t.Fatalf("Error unmarshaling modified YAML: %v", err)
	}

	if !reflect.DeepEqual(modifiedData, expectedData) {
		t.Fatalf("Expected modified YAML: %v, got: %v", expectedData, modifiedData)
	}
}
