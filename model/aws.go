package model

import "context"

// AWS defines the interface required to operate between packages
type AWS interface {
	GetCertificateByTag(ctx context.Context, key, value string) (*Certificate, error)
}
