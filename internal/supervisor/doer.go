// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Doer describes an action to be done.
type Doer interface {
	Do() error
	Shutdown()
}

// MultiDoer is a slice of doers.
type MultiDoer struct {
	doers  []Doer
	logger log.FieldLogger
}

func NewMultiDoer(logger log.FieldLogger) MultiDoer {
	return MultiDoer{
		logger: logger,
		doers:  make([]Doer, 0),
	}
}

// Do executes each doer in turn, returning the first error.
func (md MultiDoer) Do() error {
	var doerFailed bool
	for _, doer := range md.doers {
		err := doer.Do()
		if err != nil {
			doerFailed = true
			md.logger.WithError(err).Error("doer failed")
		}
	}

	if doerFailed {
		return fmt.Errorf("doers failed, check previous logs for details")
	}

	return nil
}

// Shutdown tells each doer to perform shutdown tasks.
func (md MultiDoer) Shutdown() {
	for _, doer := range md.doers {
		doer.Shutdown()
	}
}

// Append appends a doer to the list.
func (md *MultiDoer) Append(doers ...Doer) {
	md.doers = append(md.doers, doers...)
}
