package model

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
)

// EnvVar contains the value source for a given environement variable.
type EnvVar struct {
	Value     string               `json:"value"`
	ValueFrom *corev1.EnvVarSource `json:"value_from,omitempty"`
}

// ToJSON converts the EnvVarMap to a JSON object represented as a
// []byte
func (e *EnvVarMap) ToJSON() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return []byte(nil), err
	}

	return b, nil
}

// EnvVarFromJSON creates a EnvVarMap from the JSON represented as a
// []byte
func EnvVarFromJSON(raw []byte) (*EnvVarMap, error) {
	e := &EnvVarMap{}
	err := json.Unmarshal(raw, e)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// EnvVarMap is a map of multiple env var names to their values.
type EnvVarMap map[string]EnvVar

// ToEnvList returns a list of standard corev1.EnvVars
func (e *EnvVarMap) ToEnvList() []*corev1.EnvVar {
	var envList []*corev1.EnvVar

	for name, env := range *e {
		envList = append(envList, &corev1.EnvVar{
			Name:      name,
			Value:     env.Value,
			ValueFrom: env.ValueFrom,
		})
	}

	return envList
}
