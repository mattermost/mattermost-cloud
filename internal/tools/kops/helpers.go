// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package kops

import (
	"errors"
	"time"
)

// WaitForKubernetesReadiness will poll a given kubernetes cluster at a regular
// interval for it to become ready. If the cluster fails to become ready before
// the provided timeout then an error will be returned.
func (c *Cmd) WaitForKubernetesReadiness(dns string, timeout int) error {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return errors.New("timed out waiting for k8s cluster to become ready")
		default:
			err := c.ValidateCluster(dns, true)
			if err == nil {
				return nil
			}
			time.Sleep(5 * time.Second)
		}
	}
}
