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
	prevL2, prevR2 float32
	reverse bool
}

func (t *Task) Enter() {
	t.platform.SetVelocity(0, 0)
	t.platform.SetServos(0.0, 0.0)
	t.platform.EnableServos(true, true)
}

func (t *Task) Exit() {
	t.platform.SetVelocity(0, 0)
	t.platform.SetServos(0.0, 0.0)
	t.platform.EnableServos(false, false)
}

func (t *Task) Tick(buttons input.ButtonState) {
	maxSpeed := t.platform.GetMaxVelocity()
	maxW := t.platform.GetMaxOmega()
	a, b := t.input.GetSticks()

	if a != t.prevA || b != t.prevB {
		//t.platform.SetVelocity(a * maxSpeed, b * maxSpeed)
		speed, w := a * maxSpeed, b * maxW
		if t.reverse {
			speed = -speed
		}
		t.platform.SetArc(speed, w)
	}
	t.prevA = a
	t.prevB = b

	if buttons[input.Triangle] == input.Pressed {
		t.reverse = !t.reverse
		buttons[input.Triangle] = input.None
	}

	l2, r2 := t.input.GetTriggers()
	if l2 != t.prevL2 || r2 != t.prevR2 {
		t.platform.SetServos(l2, r2)
	}
	t.prevL2, t.prevR2 = l2, r2
}

func NewTask(ip *input.Collector, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		input: ip,
	}
}
