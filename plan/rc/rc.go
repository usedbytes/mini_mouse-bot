// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package rc

import (
	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
)

const TaskName = "rc"

type Task struct {
	platform *base.Platform
	input *input.Collector
}

func (t *Task) Tick() {
	maxSpeed := t.platform.GetMaxVelocity()
	a, b := t.input.GetSticks()
	t.platform.SetVelocity(a * maxSpeed, b * maxSpeed)
}

func NewTask(ip *input.Collector, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		input: ip,
	}
}
