package clicontext

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-cloud/internal/auth"
)

type ContextKeyServerURL struct{}

type CLIContext struct {
	AuthData  *auth.AuthorizationResponse `json:"auth_data"`
	ClientID  string                      `json:"client_id"`
	OrgURL    string                      `json:"org_url"`
	ServerURL string                      `json:"server_url"`
	Alias     string                      `json:"alias"`
}

type Contexts struct {
	CurrentContext string                `json:"current_context"`
	Contexts       map[string]CLIContext `json:"contexts"`
}

func (c *Contexts) Current() *CLIContext {
	context, ok := c.Contexts[c.CurrentContext]
	if !ok {
		return nil
	}

	return &context
}

func (c *Contexts) UpdateContext(contextName string, authData *auth.AuthorizationResponse, clientID, orgURL, alias, serverURL string) error {
	c.Contexts[contextName] = CLIContext{
		AuthData:  authData,
		ClientID:  clientID,
		OrgURL:    orgURL,
		Alias:     alias,
		ServerURL: serverURL,
	}

	return WriteContexts(c)
}

func contextFileDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".cloud")
}

func contextFileSystemPath() string {
	contextDir := contextFileDirectory()

	return filepath.Join(contextDir, "contexts.json")
}

func bootstrapFirstContext() Contexts {
	contexts := Contexts{
		Contexts: make(map[string]CLIContext, 1),
	}
	contexts.CurrentContext = "local"
	contexts.Contexts["local"] = CLIContext{
		ServerURL: "http://localhost:8075",
		Alias:     "local",
	}
	return contexts
}

func ReadContexts() (*Contexts, error) {
	contextFilePath := contextFileSystemPath()

	var contextsData Contexts
	if _, statErr := os.Stat(contextFilePath); errors.Is(statErr, os.ErrNotExist) {
		contextsData = bootstrapFirstContext()
		wErr := WriteContexts(&contextsData)
		return &contextsData, wErr
	}

	data, err := os.ReadFile(contextFilePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &contextsData)
	if err != nil {
		return nil, err
	}

	return &contextsData, nil
}

func WriteContexts(contexts *Contexts) error {
	// create the .cloud directory if it doesn't exist
	cloudDir := contextFileDirectory()
	if _, err := os.Stat(cloudDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(cloudDir, 0700)
		if err != nil {
			return err
		}
	}

	contextFilePath := contextFileSystemPath()
	file, err := os.Create(contextFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(contexts)
	if err != nil {
		return err
	}

	return nil
}
