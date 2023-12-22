package argocd

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAddClusterIDLabel(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "testyaml")
	if err != nil {
		t.Fatalf("Error creating temporary YAML file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	testYAML := []byte(`
applications:
  - name: tempo
    cluster_labels:
      - cluster-type: cnc
  - name: loki
    cluster_labels:
      - cluster-type: cnc
`)
	if _, err := tmpfile.Write(testYAML); err != nil {
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

	AddClusterIDLabel(data, "loki", "cluster1", nil)

	expectedModifiedYAML := []byte(`
applications:
	- name: tempo
	cluster_labels:
		- cluster-type: cnc
	- name: loki
	cluster_labels:
		- cluster-type: cnc
		- cluster-id: cluster1
`)
	modifiedYAML, err := yaml.Marshal(&data)
	if err != nil {
		t.Fatalf("Error marshalling YAML: %v", err)
	}

	if !reflect.DeepEqual(modifiedYAML, expectedModifiedYAML) {
		t.Fatalf("Expected modified YAML: %v, got: %v", expectedModifiedYAML, modifiedYAML)
	}
}

func TestRemoveClusterIDLabel(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "testyaml")
	if err != nil {
		t.Fatalf("Error creating temporary YAML file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	testYAML := []byte(`
applications:
- name: tempo
	cluster_labels:
	- cluster-type: cnc
	- cluster-id: ID1
- name: loki
	cluster_labels:
	- cluster-type: cnc
	- cluster-id: ID2
`)
	if _, err := tmpfile.Write(testYAML); err != nil {
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

	RemoveClusterIDLabel(data, "loki", "ID2", nil)

	expectedModifiedYAML := []byte(`
applications:
- name: tempo
	cluster_labels:
	- cluster-type: cnc
	- cluster-id: ID1
- name: loki
	cluster_labels:
	- cluster-type: cnc
`)
	modifiedYAML, err := yaml.Marshal(&data)
	if err != nil {
		t.Fatalf("Error marshalling YAML: %v", err)
	}

	if !reflect.DeepEqual(modifiedYAML, expectedModifiedYAML) {
		t.Fatalf("Expected modified YAML: %v, got: %v", expectedModifiedYAML, modifiedYAML)
	}
}
