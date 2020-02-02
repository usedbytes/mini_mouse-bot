// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package square

import (
	"image/color"
	"math"

	"github.com/usedbytes/thunk-bot/base"
	"github.com/usedbytes/thunk-bot/interface/input"
	"github.com/usedbytes/thunk-bot/model"
	"github.com/usedbytes/thunk-bot/plan/heading"
)

const TaskName = "square"

type Task struct {
	platform *base.Platform
	model *model.Model
	heading *heading.Task
	running bool
	turning bool
	moving bool
	dir float32
}

func (t *Task) Enter() {
	t.dir = 0.0
	t.running = false
	t.turning = true

	t.model.ResetOrientation();
	t.heading.SetHeading(t.dir);
}

func (t *Task) Exit() {
	t.platform.SetVelocity(0, 0)
}

func (t *Task) Tick(buttons input.ButtonState) {
	if buttons[input.Cross] == input.Pressed {
		buttons[input.Cross] = input.None
		if t.running {
			t.platform.SetVelocity(0, 0)
		}
		t.running = !t.running
	}

	if (!t.running) {
		return
	}

	if t.turning {
		t.heading.Tick(buttons)
		if !t.heading.OnCourse {
			return
		}
		t.turning = false
		t.platform.ControlledMove(200, t.platform.GetMaxVelocity() * 0.7)
		t.moving = true

		t.dir += math.Pi / 2
		t.heading.SetHeading(t.dir)
		return
	}

	if t.moving {
		if t.platform.Moving() {
			return
		}
		t.platform.SetVelocity(0, 0)
		t.moving = false
		t.turning = false
		return
	}

	t.turning = true
}

func (t *Task) Color() color.Color {
	return color.NRGBA{ 0xf4, 0x42, 0x86, 0x80 }
}

func NewTask(m *model.Model, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		model: m,
		heading: heading.NewTask(m, pl),
	}
}
