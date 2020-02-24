package model

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// EnvVar contains the value source for a given environment variable.
type EnvVar struct {
	Value     string               `json:"value,omitempty"`
	ValueFrom *corev1.EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarMap is a map of multiple env var names to their values.
type EnvVarMap map[string]EnvVar

// Validate returns wheather the env var has valid value configuration.
func (e *EnvVar) Validate() error {
	if len(e.Value) == 0 && e.ValueFrom == nil {
		return errors.New("no Value or ValueFrom is defined")
	}

	return nil
}

// Validate returns wheather the env var map has valid value configuration.
func (em *EnvVarMap) Validate() error {
	for name, env := range *em {
		err := env.Validate()
		if err != nil {
			return errors.Wrapf(err, "invalid env var %s", name)
		}
	}

	return nil
}

// ToEnvList returns a list of standard corev1.EnvVars
func (em *EnvVarMap) ToEnvList() []corev1.EnvVar {
	envList := []corev1.EnvVar{}

	for name, env := range *em {
		envList = append(envList, corev1.EnvVar{
			Name:      name,
			Value:     env.Value,
			ValueFrom: env.ValueFrom,
		})
	}

	return envList
}
