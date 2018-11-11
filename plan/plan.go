// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package plan

import (
	"fmt"

	"github.com/usedbytes/mini_mouse/bot/interface/input"
)

type Task interface {
	Tick(buttons input.ButtonState)
}

type Planner struct {
	current string
	tasks map[string]Task
}

func (p *Planner) Tick(buttons input.ButtonState) {
	// TODO: Do things which are irrespective of task

	if p.current == "" {
		return
	}

	p.tasks[p.current].Tick(buttons)
}

func (p *Planner) SetTask(name string) error {
	if _, ok := p.tasks[name]; !ok {
		return fmt.Errorf("Unknown task '%s'", name)
	}

	// TODO: Stop current task

	p.current = name
	return nil
}

func (p *Planner) AddTask(name string, task Task) error {
	if _, ok := p.tasks[name]; ok {
		return fmt.Errorf("Duplicate task name '%s'", name)
	}

	p.tasks[name] = task
	return nil
}

func NewPlanner() *Planner {
	return &Planner{
		tasks: make(map[string]Task),
	}
}
