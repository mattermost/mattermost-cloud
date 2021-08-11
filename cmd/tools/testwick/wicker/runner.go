// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package testwick

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// StepFunc the func to run for a step
type StepFunc func(*TestWicker, context.Context) error

// Step is a single step in a workflow.
type Step struct {
	Name string
	Func StepFunc
}

// Workflow is steps based workflow.
type Workflow struct {
	Steps  []Step
	logger *logrus.Logger
}

// NewWorkflow factory method
func NewWorkflow(logger *logrus.Logger) *Workflow {
	return &Workflow{
		logger: logger,
	}
}

// AddStep add step to the Workflow.
func (w *Workflow) AddStep(step ...Step) *Workflow {
	w.Steps = append(w.Steps, step...)
	return w
}

// Run will execute the set of commands of a recipe
func (w *Workflow) Run(ctx context.Context, testWicker *TestWicker) error {
	for _, step := range w.Steps {
		w.logger.WithField("Step", step.Name).Info("Running")
		if err := step.Func(testWicker, ctx); err != nil {
			return errors.Wrapf(err, "Step %s failed", step.Name)
		}
		w.logger.WithField("Step", step.Name).Info("Finished Succefully")
	}
	return nil
}
