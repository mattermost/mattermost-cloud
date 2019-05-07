package api_test

type mockSupervisor struct {
}

func (s *mockSupervisor) Do() error {
	return nil
}
