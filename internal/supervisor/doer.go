package supervisor

// Doer describes an action to be done.
type Doer interface {
	Do() error
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
