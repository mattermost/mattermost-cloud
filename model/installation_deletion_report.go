package model

import "time"

// DeletionPendingReport is a collection of configurable time cutoffs that are
// used to summarize installation deletion times. There is also an Overflow
// value that counts installations that fall outside all provided time cutoffs.
// Examples of DeletionPendingReport cutoffs:
// 1. Every day for a week
// 2, Every hour for a day
// 3. Within 1 hour, 1 day, 1 month, 1 year
type DeletionPendingReport struct {
	Cutoffs  []*DeletionPendingTimeCutoff
	Overflow int
}

// DeletionPendingTimeCutoff is a single deletion time cutoff.
type DeletionPendingTimeCutoff struct {
	Name   string
	Millis int64
	Count  int
}

// NewCutoff adds a new deletion time cutoff.
func (dpr *DeletionPendingReport) NewCutoff(name string, cutoffTime time.Time) {
	dpr.Cutoffs = append(dpr.Cutoffs, &DeletionPendingTimeCutoff{
		Name:   name,
		Millis: GetMillisAtTime(cutoffTime),
	})
}

// Count increases the Count value of the first Cutoff that is greater than the
// provided Millis.
func (dpr *DeletionPendingReport) Count(millis int64) {
	for _, cutoff := range dpr.Cutoffs {
		if cutoff.Millis > millis {
			cutoff.Count++
			return
		}
	}

	// The time passed in is greater than all cutoffs so add it to the overflow.
	dpr.Overflow++
}
