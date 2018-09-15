// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package plan

import (
	"fmt"
)

type Task interface {
	Tick()
}

type Planner struct {
	current string
	tasks map[string]Task
}

func (p *Planner) Tick() {
	// TODO: Do things which are irrespective of task

	if p.current == "" {
		return
	}

	p.tasks[p.current].Tick()
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
