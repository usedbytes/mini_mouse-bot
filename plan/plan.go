// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package plan

import (
	"image/color"
	"fmt"

	"github.com/usedbytes/thunk-bot/base"
	"github.com/usedbytes/thunk-bot/interface/input"
	"github.com/usedbytes/thunk-bot/interface/menu"
)

type Task interface {
	Tick(buttons input.ButtonState)
	Color() color.Color
}

type EnterExitTask interface {
	Task
	Enter()
	Exit()
}

type Planner struct {
	current Task
	tasks map[string]Task
	idleTask Task
	mainMenu *menu.Menu
	platform *base.Platform
}

func (p *Planner) Tick(buttons input.ButtonState) {
	if p.current == p.idleTask {
		p.mainMenu.Tick(buttons)
	}

	if p.current == nil {
		return
	}

	p.current.Tick(buttons)
}

func (p *Planner) SetTask(name string) error {
	if _, ok := p.tasks[name]; !ok {
		return fmt.Errorf("Unknown task '%s'", name)
	}

	exit, ok := p.current.(EnterExitTask)
	if ok {
		exit.Exit()
	}

	fmt.Println("Task", name)
	p.current = p.tasks[name]

	p.platform.SetLEDColor(p.current.Color())
	enter, ok := p.current.(EnterExitTask)
	if ok {
		enter.Enter()
	}
	return nil
}

func (p *Planner) AddTask(name string, task Task, d menu.Direction) error {
	if _, ok := p.tasks[name]; ok {
		return fmt.Errorf("Duplicate task name '%s'", name)
	}

	p.tasks[name] = task
	if d != menu.None {
		p.mainMenu.AddItem(d, task.Color(), func() { p.SetTask(name) })
	} else {
		p.idleTask = task
	}

	return nil
}

func (p *Planner) CurrentTask() Task {
	return p.current
}

func NewPlanner(p *base.Platform) *Planner {
	return &Planner{
		tasks: make(map[string]Task),
		platform: p,
		mainMenu: menu.NewMenu(p),
	}
}
