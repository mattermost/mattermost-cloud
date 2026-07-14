# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

- Add durable project-specific notes here as they are discovered through real work.

## Pod scheduling / affinity

- The provisioner OWNS the Mattermost CR's scheduling: `ensureScheduling` in
  `internal/provisioner/cluster_installation_provisioner.go` overwrites
  `mattermost.Spec.Scheduling.Affinity` (via `generateAffinityConfig`) and
  `mattermost.Spec.ResourceLabels` (via `clusterInstallationStableLabels`) on EVERY
  create/update reconcile. Hand-patching the CR or the Deployment is reverted on the next
  reconcile — durable scheduling changes must live in the provisioner. The operator sets no
  default affinity; it passes `spec.scheduling.affinity` straight through.
- Pod labels come from `Spec.ResourceLabels` (the operator seeds pod labels from it), so any
  label added in `clusterInstallationStableLabels` lands on the pods and is selectable by
  anti-affinity.
- `model.Installation` does NOT carry its annotations (separate join). Fetch them via
  `store.GetAnnotationsForInstallation(installation.ID)` at the reconcile call sites (see
  `resolveNodeSeparation`). The `separate-nodes` annotation is the opt-in for cross-installation
  node separation (SEC-9253).
