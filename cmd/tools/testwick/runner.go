package main

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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

// New factory method
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

// Execute will execute the set of commands of a recipe
func (w *Workflow) Run(testWicker *TestWicker, ctx context.Context) error {
	for _, step := range w.Steps {
		w.logger.WithField("Step", step.Name).Info("Running")
		if err := step.Func(testWicker, ctx); err != nil {
			return errors.Wrapf(err, "Step %s failed", step.Name)
		}
		w.logger.WithField("Step", step.Name).Info("Finished Succefully")
	}
	return nil
}
