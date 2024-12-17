// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"sort"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	VolumeTypeSecret = "secret"

	volumeNameMinLen        = 3
	volumeNameMaxLen        = 64
	volumeNameAllowedFormat = "volumes must start with a letter and can contain only lowercase letters, numbers or '_', '-' characters"
)

var volumeNameRegex = regexp.MustCompile("^[a-z]+[a-z0-9_-]*$")

// Volume contains metadata on a k8s volume.
type Volume struct {
	Type          string
	BackingSecret string
	MountPath     string
	ReadOnly      bool
}

// VolumeMap is a map of multiple Volume names to their values.
type VolumeMap map[string]Volume

// Validate returns wheather Volume has valid value configuration.
func (v *Volume) Validate() error {
	if v.Type != VolumeTypeSecret {
		return errors.Errorf("%s is an invalid volume type", v.Type)
	}
	if len(v.MountPath) == 0 {
		return errors.New("mount path is empty")
	}

	return nil
}

// Validate returns wheather the VolumeMap has valid value configuration.
func (vm *VolumeMap) Validate() error {
	for name, env := range *vm {
		err := env.Validate()
		if err != nil {
			return errors.Wrapf(err, "invalid volume %s", name)
		}
	}

	return nil
}

// Add applies a CreateInstallationVolumeRequest to a VolumeMap.
func (vm VolumeMap) Add(request *CreateInstallationVolumeRequest) error {
	if _, ok := vm[request.Name]; ok {
		return errors.Errorf("cannot create new volume, %s already exists", request.Name)
	}
	if len(request.Name) < volumeNameMinLen || len(request.Name) > volumeNameMaxLen {
		return errors.Errorf("volume '%s' is invalid: volume names must be between %d and %d characters long", request.Name, volumeNameMinLen, volumeNameMaxLen)
	}
	if !volumeNameRegex.MatchString(request.Name) {
		return errors.Errorf("volume '%s' is invalid: %s", request.Name, volumeNameAllowedFormat)
	}
	for name, existing := range vm {
		if existing.MountPath == request.Volume.MountPath {
			return errors.Errorf("mount path %s conflicts with volume %s", existing.MountPath, name)
		}
	}

	vm[request.Name] = *request.Volume

	return nil
}

// Patch applies a PatchInstallationVolumeRequest to a VolumeMap.
func (vm VolumeMap) Patch(request *PatchInstallationVolumeRequest, volumeName string) (string, error) {
	volume, ok := vm[volumeName]
	if !ok {
		return "", errors.Errorf("cannot update volume %s as it doesn't exist", volumeName)
	}

	if request.MountPath != nil {
		for existingName, existing := range vm {
			if existing.MountPath == *request.MountPath && existingName != volumeName {
				return "", errors.Errorf("mount path %s conflicts with volume %s", existing.MountPath, existingName)
			}
		}
		volume.MountPath = *request.MountPath
	}
	if request.ReadOnly != nil {
		volume.ReadOnly = *request.ReadOnly
	}

	vm[volumeName] = volume

	return volume.BackingSecret, nil
}

// ToCoreV1Volumes returns a list of standard corev1.Volume.
func (vm *VolumeMap) ToCoreV1Volumes() []corev1.Volume {
	volumes := []corev1.Volume{}

	for name, vol := range *vm {
		volumes = append(volumes, corev1.Volume{
			Name:         name,
			VolumeSource: vol.CoreV1VolumeSource(name),
		})
	}

	// To retain consistent order of volumes we sort the slice by name.
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	return volumes
}

// ToCoreV1VolumeMounts returns a list of standard corev1.VolumeMount.
func (vm *VolumeMap) ToCoreV1VolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}

	for name, vol := range *vm {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      name,
			ReadOnly:  vol.ReadOnly,
			MountPath: vol.MountPath,
		})
	}

	// To retain consistent order of volume mounts we sort the slice by name.
	sort.Slice(volumeMounts, func(i, j int) bool {
		return volumeMounts[i].Name < volumeMounts[j].Name
	})

	return volumeMounts
}

// CoreV1VolumeSource returns a corev1.VolumeSource for a given volume.
func (v *Volume) CoreV1VolumeSource(name string) corev1.VolumeSource {
	volumeSource := corev1.VolumeSource{}

	switch v.Type {
	case VolumeTypeSecret:
		volumeSource.Secret = &corev1.SecretVolumeSource{
			SecretName: name,
		}
	}

	return volumeSource
}

func (vm VolumeMap) Value() (driver.Value, error) {
	if vm == nil {
		return nil, nil
	}

	return json.Marshal(vm)
}

func (vm *VolumeMap) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.Errorf("could not assert type of VolumeMap (value: %+v)", src)
	}

	var i VolumeMap
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}
	*vm = i
	return nil
}
