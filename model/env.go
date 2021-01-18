// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"sort"

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

// HasValue returns wheather the env var has any value configuration or not.
func (e *EnvVar) HasValue() bool {
	return len(e.Value) != 0 || e.ValueFrom != nil
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

// ClearOrPatch takes a new EnvVarMap and patches changes into the existing
// EnvVarMap with the following logic:
//  - If the new EnvVarMap is empty, clear the existing EnvVarMap completely.
//  - If the new EnvVarMap is not empty, apply normal patch logic.
func (em *EnvVarMap) ClearOrPatch(new *EnvVarMap) bool {
	if *em == nil {
		if len(*new) == 0 {
			return false
		}

		*em = *new
		return true
	}

	if len(*new) == 0 {
		orginalEmpty := len(*em) != 0
		*em = nil

		return orginalEmpty
	}

	return em.Patch(*new)
}

// Patch takes a new EnvVarMap and patches changes into the existing EnvVarMap
// with the following logic:
//  - If the new EnvVar has the same key as an old EnvVar, update the value.
//  - If the new EnvVar is a new key, add the EnvVar.
//  - If the new EnvVar has no value(is blank), clear the old EnvVar if there
//    was one.
func (em EnvVarMap) Patch(new EnvVarMap) bool {
	if new == nil {
		return false
	}

	var wasPatched bool
	for newName, newEnv := range new {
		if oldEnv, ok := em[newName]; ok {
			// This EnVar exists already. Delete it or update it if the patch
			// value is different.
			if !newEnv.HasValue() {
				wasPatched = true
				delete(em, newName)
				continue
			}

			if oldEnv != newEnv {
				wasPatched = true
				em[newName] = newEnv
			}
		} else {
			// This EnvVar doesn't exist in the original map. Add it if the
			// patch has a value.
			if newEnv.HasValue() {
				wasPatched = true
				em[newName] = newEnv
			}
		}
	}

	return wasPatched
}

// ToEnvList returns a list of standard corev1.EnvVars.
func (em *EnvVarMap) ToEnvList() []corev1.EnvVar {
	envList := []corev1.EnvVar{}

	for name, env := range *em {
		envList = append(envList, corev1.EnvVar{
			Name:      name,
			Value:     env.Value,
			ValueFrom: env.ValueFrom,
		})
	}

	// To retain consistent order of environment variables - sort the array.
	// Changing the order of env vars, even if they did not change will cause
	// rotation of Cluster Installation's pods.
	sort.Slice(envList, func(i, j int) bool {
		return envList[i].Name < envList[j].Name
	})

	return envList
}

// ToJSON converts the EnvVarMap to a JSON object represented as a []byte.
func (em *EnvVarMap) ToJSON() ([]byte, error) {
	return json.Marshal(em)
}

// EnvVarFromJSON creates a EnvVarMap from the JSON represented as a []byte.
func EnvVarFromJSON(raw []byte) (*EnvVarMap, error) {
	e := &EnvVarMap{}
	err := json.Unmarshal(raw, e)
	if err != nil {
		return nil, err
	}
	return e, nil
}
