// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

// Doer describes an action to be done.
type Doer interface {
	Do() error
	Shutdown()
}

// MultiDoer is a slice of doers.
type MultiDoer []Doer

// Do executes each doer in turn, returning the first error.
func (md MultiDoer) Do() error {
	for _, doer := range md {
		err := doer.Do()
		if err != nil {
			return err
		}
	}

	return nil
}

// Shutdown tells each doer to perform shutdown tasks.
func (md MultiDoer) Shutdown() {
	for _, doer := range md {
		doer.Shutdown()
	}
}
