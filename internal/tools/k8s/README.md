# Kubernetes Package

This package contains the resources necessary to create a kubernetes client that is capable of interacting with Mattermost Custom Resources such as a cluster installation.

To do this generated code is leveraged. This code is generated with the help of: https://github.com/kubernetes/code-generator

The generator takes the files found in `pkg/apis/mattermost.com/v1alpha1` and creates the necessary code to add this custom resource to the go kubernetes client.

To regenerate this code, do the following:

```
go get github.com/kubernetes/code-generator
cd code-generator
git checkout kubernetes-1.14.0
$(go env GOPATH)/src/k8s.io/code-generator/generate-groups.sh all   github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg/apis   mattermost.com:v1alpha1
```

##### Note 1: if be sure to checkout the correct tag that targets the kubernetes version you will be working with.
##### Note 2: change the mattermost version number as necessary.

This will generate additional code in the `pkg` directories for the clientset, listers, etc.