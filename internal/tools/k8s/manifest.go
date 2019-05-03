package k8s

import "path"

// ManifestFile is a file containing kubernetes resources.
type ManifestFile struct {
	Path            string
	DeployNamespace string
}

// Basename returns the base filename of the manifest file.
func (f *ManifestFile) Basename() string {
	return path.Base(f.Path)
}
