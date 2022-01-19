// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// +build e2e

package workflow

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost-cloud/e2e/pkg/eventstest"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// StepFunc defines the structure of functions called as steps.
type StepFunc func(ctx context.Context) error

// Step is a single step in a workflow.
type Step struct {
	Name              string
	Func              StepFunc
	Done              bool
	DependsOn         []string
	GetExpectedEvents func() []eventstest.EventOccurrence
}

// NewWorkflow creates new Workflow.
func NewWorkflow(steps []*Step) *Workflow {
	stepsMap := make(map[string]*Step, len(steps))
	workflow := Workflow{stepsMap}

	return workflow.AddStep(steps...)
}

// Workflow is steps based workflow.
type Workflow struct {
	Steps map[string]*Step
}

// AddStep add step to the Workflow.
func (w *Workflow) AddStep(step ...*Step) *Workflow {
	for i, s := range step {
		w.Steps[s.Name] = step[i]
	}
	return w
}

// RunWorkflow runs Workflow considering all dependencies.
func RunWorkflow(workflow *Workflow, logger logrus.FieldLogger) error {
	runner := &runner{
		workflow: *workflow,
		queue:    make([]Step, 0, len(workflow.Steps)),
		inQueue:  make(map[string]bool, len(workflow.Steps)),
	}

	err := runner.makeQueue()
	if err != nil {
		return errors.Wrap(err, "failed to make queue")
	}

	ctx := context.Background()

	for _, step := range runner.queue {
		if step.Done {
			logger.Infof("Step %s marked as done, skipping", step.Name)
			continue
		}

		logrus.Infof("Running step: %s", step.Name)
		err := step.Func(ctx)
		if err != nil {
			return errors.Wrapf(err, "step %s failed", step.Name)
		}
		logger.Infof("Step %s finished successfully", step.Name)
		workflow.Steps[step.Name].Done = true
	}

	logger.Infof("Workflow finished")

	return nil
}

type runner struct {
	queue    []Step
	workflow Workflow
	inQueue  map[string]bool
}

func (l *runner) makeQueue() error {
	for _, step := range l.workflow.Steps {
		err := l.addToQueue(step)
		if err != nil {
			return errors.Wrap(err, "error while adding step to queue")
		}
	}

	return nil
}

func (l *runner) addToQueue(step *Step) error {
	if step == nil {
		return fmt.Errorf("cannot add nil step to queue")
	}
	if l.inQueue[step.Name] {
		return nil
	}

	for _, d := range step.DependsOn {
		dep, found := l.workflow.Steps[d]
		if !found {
			return fmt.Errorf("step %q not found in workflow but step %q depends on it", d, step.Name)
		}
		err := l.addToQueue(dep)
		if err != nil {
			return err
		}
	}

	l.queue = append(l.queue, *step)
	l.inQueue[step.Name] = true

	return nil
}
