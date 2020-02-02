// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package rc

import (
	"image/color"

	"github.com/usedbytes/thunk-bot/base"
	"github.com/usedbytes/thunk-bot/interface/input"
)

const TaskName = "rc"

type Task struct {
	platform *base.Platform
	input *input.Collector

	prevA, prevB float32
	prevL2, prevR2 float32
	reverse bool
	boost base.Boost
}

func (t *Task) Enter() {
	t.platform.SetVelocity(0, 0)
	t.platform.SetServos(0.0, 0.0)
	t.platform.EnableServos(true, true)
	t.reverse = false
}

func (t *Task) Exit() {
	t.platform.SetVelocity(0, 0)
	t.platform.SetServos(0.0, 0.0)
	t.platform.EnableServos(false, false)

	t.platform.SetBoost(base.BoostNone)
}

func (t *Task) Tick(buttons input.ButtonState) {
	boost := base.BoostNone
	if buttons[input.L1] == input.Held || buttons[input.L1] == input.LongPress {
		boost = base.BoostSlow
	} else if buttons[input.R1] == input.Held || buttons[input.R1] == input.LongPress {
		boost = base.BoostFast
	}

	if boost != t.boost {
		t.platform.SetBoost(boost)
	}

	maxSpeed := t.platform.GetMaxVelocity()

	maxW := t.platform.GetMaxOmega()
	a, b := t.input.GetSticks()

	update := a != t.prevA || b != t.prevB || boost != t.boost
	t.boost = boost

	if update {
		//t.platform.SetVelocity(a * maxSpeed, b * maxSpeed)
		speed, w := a * maxSpeed, b * maxW
		if t.reverse {
			speed = -speed
		}
		t.platform.SetArc(speed, w)
	}
	t.prevA = a
	t.prevB = b

	if buttons[input.Square] == input.Pressed {
		t.reverse = !t.reverse
		buttons[input.Square] = input.None
	}

	l2, r2 := t.input.GetTriggers()
	if l2 != t.prevL2 || r2 != t.prevR2 {
		t.platform.SetServos(l2, r2)
	}
	t.prevL2, t.prevR2 = l2, r2
}

func (t *Task) Color() color.Color {
	return color.NRGBA{ 0x00, 0xff, 0x00, 0x80 }
}

func NewTask(ip *input.Collector, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		input: ip,
	}
}
