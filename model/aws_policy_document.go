package model

import "encoding/json"

const (
	// PolicyDocumentVersion supplies the default version for creating policy documents.
	PolicyDocumentVersion = "2012-10-17"

	// PolicyStatementEffectAllow supplies the default effect name for allowing resource actions
	// when creating a policy statement.
	PolicyStatementEffectAllow = "Allow"

	// PolicyStatementAllResources supplies the default value for including all resource types in the
	// policy statement.
	PolicyStatementAllResources = "*"
)

// PolicyDocument creates a policy document.
type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

// StatementEntry creates a statement in the policy document.
type StatementEntry struct {
	Effect    string
	Principal map[string]string
	Action    []string
	Resource  string
}

// Marshal returns json text bytes of the policy object.
func (p *PolicyDocument) Marshal() ([]byte, error) {
	return json.Marshal(*p)
}
