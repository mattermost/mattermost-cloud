package api_test

type mockTerraformCmd struct {
	outputs map[string]string
}

func newMockTerraformCmd() *mockTerraformCmd {
	return &mockTerraformCmd{
		outputs: make(map[string]string),
	}
}

func (m *mockTerraformCmd) Init() error {
	return nil
}

func (m *mockTerraformCmd) Apply() error {
	return nil
}

func (m *mockTerraformCmd) ApplyTarget(target string) error {
	return nil
}

func (m *mockTerraformCmd) MockOutput(variable, value string) {
	m.outputs[variable] = value
}

func (m *mockTerraformCmd) Output(variable string) (string, error) {
	return m.outputs[variable], nil
}

func (m *mockTerraformCmd) Destroy() error {
	return nil
}

func (m *mockTerraformCmd) Close() error {
	return nil
}
