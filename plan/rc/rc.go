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

	prevA, prevB float32
}

func (t *Task) Tick() {
	maxSpeed := t.platform.GetMaxVelocity()
	maxW := t.platform.GetMaxOmega()
	a, b := t.input.GetSticks()

	if a != t.prevA || b != t.prevB {
		//t.platform.SetVelocity(a * maxSpeed, b * maxSpeed)
		t.platform.SetArc(a * maxSpeed, b * maxW)
	}
	t.prevA = a
	t.prevB = b
}

func NewTask(ip *input.Collector, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		input: ip,
	}
}
