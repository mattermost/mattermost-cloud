// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package components

// Contains returns true if collections contains at least one matching element.
func Contains(collection []string, toFind string) bool {
	for _, elem := range collection {
		if toFind == elem {
			return true
		}
	}
	return false
}
