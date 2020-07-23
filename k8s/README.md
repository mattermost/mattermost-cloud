# Kubernetes Package

This package contains the resources necessary to create a client capable of interacting with different kubernetes clusters.

The clientset for the Mattermost `ClusterInstallation` custom resource now resides in the [mattermost-operator](https://github.com/mattermost/mattermost-operator) repository. This provides a single source of truth for `ClusterInstallation` objects. To change the clientset used here, first update the code in `mattermost-operator` and then import the changes.

More information on doing this can be found in the [mattermost-operator README](https://github.com/mattermost/mattermost-operator/blob/master/README.md).