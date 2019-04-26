package k8s

import "path"

// ManifestFile is a file containing kubernetes resources.
type ManifestFile struct {
	Name            string
	Directory       string
	DeployNamespace string
}

// FQN provides the fully qualified name where the manifest is located.
func (f *ManifestFile) FQN() string {
	return path.Join(f.Directory, f.Name)
}
