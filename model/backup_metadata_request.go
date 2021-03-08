package model

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// BackupRequest represents request for installation backup.
type BackupRequest struct {
	InstallationID string
}

// NewBackupRequestFromReader will create a BackupMetadata from an
// io.Reader with JSON data.
func NewBackupRequestFromReader(reader io.Reader) (*BackupRequest, error) {
	var backupRequest BackupRequest
	err := json.NewDecoder(reader).Decode(&backupRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode backup request")
	}

	return &backupRequest, nil
}

// GetBackupsMetadataRequest describes the parameters to request a list of installation backups.
type GetBackupsMetadataRequest struct {
	InstallationID        string
	ClusterInstallationID string
	State                 string
	Page                  int
	PerPage               int
	IncludeDeleted        bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetBackupsMetadataRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("installation", request.InstallationID)
	q.Add("cluster_installation", request.ClusterInstallationID)
	q.Add("state", request.State)
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
	u.RawQuery = q.Encode()
}
