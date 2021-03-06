///////////////////////////////////////////////////////////////////////
// Copyright (C) 2016 VMware, Inc. All rights reserved.
// -- VMware Confidential
///////////////////////////////////////////////////////////////////////

package task

import (
	"context"

	"github.com/casualjim/rabbit/joint"
)

//StepInfo has the information of a step
type StepInfo struct {
	State State
	Name  string
}

//Step is one step of a Task, it can be one single operation or has sequential/parallel steps
type Step interface {
	Run(context.Context) (context.Context, *joint.Error)
	Rollback(context.Context) (context.Context, *joint.Error)
	GetInfo() StepInfo
	GetSteps() []Step
}

//GenericStep is a generic Step
type GenericStep struct {
	StepInfo
	Steps []Step

	//The successFn signature takes the parent Step, the context and the child step
	//so that if any information from child step is needed, it can be done in this function.
	//Same reason for failFn.
	successFn func(Step, context.Context, Step)
	failFn    func(Step, context.Context, Step, *joint.Error)
}

func NewStepInfo(name string) StepInfo {
	return StepInfo{Name: name, State: StateNone}
}

func NewGenericStep(stepInfo StepInfo, steps ...Step) *GenericStep {
	return &GenericStep{
		StepInfo: stepInfo,
		Steps:    steps,
	}
}

///////////////////////////////////////////////////////////////////////
//Each specific step is supposed to have its own methods defined.
//Those methods on StepOpts are mostly dummy ones in case if a step does not have or
//does not need an implementation on certain methods.

func (s *GenericStep) Run(reqCtx context.Context) (context.Context, *joint.Error) {
	return reqCtx, nil
}

func (s *GenericStep) Rollback(reqCtx context.Context) (context.Context, *joint.Error) {
	return reqCtx, nil
}

func (s *GenericStep) GetInfo() StepInfo {
	return s.StepInfo
}

func (s *GenericStep) SetState(state State) {
	s.State = state
}

func (s *GenericStep) GetSteps() []Step {
	return s.Steps
}

///////////////////////////////////////////////////////////////////////
//Utility functions for Step
type StepPredicate func(Step) bool

func Filter(s Step, pred StepPredicate) []Step {
	var res []Step
	for _, step := range s.GetSteps() {
		if pred(step) {
			res = append(res, step)
		}
	}
	return res
}

func GetActiveSteps(step Step, deepest bool) []Step {
	return FindSteps(step, func(s Step) bool {
		return s.GetInfo().State == StateProcessing || s.GetInfo().State == StateRollingback
	}, deepest)
}

func FindSteps(step Step, pred StepPredicate, deepest bool) []Step {
	if !pred(step) {
		return nil
	}
	var res []Step

	switch s := step.(type) {
	case *SeqStep:
		if s.Steps == nil {
			res = append(res, step)
			return res
		}
		for _, thisStep := range s.Steps {
			if pred(thisStep) {
				if deepest {
					return FindSteps(thisStep, pred, true)
				}
				//look for the first level active steps
				res = append(res, thisStep)
			}
		}

		if deepest {
			res = append(res, step)
		}
		return res

	default:
		res = append(res, s)
		return res
	}

}
