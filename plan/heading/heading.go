// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package heading

import (
	"math"

	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/model"
)

const TaskName = "heading"

type Task struct {
	platform *base.Platform
	model *model.Model

	speed, heading float32
	OnCourse bool
}

func normalise(rads float32) float32 {
	if rads > math.Pi || rads < -math.Pi {
		rads = float32(math.Atan2(math.Sin(float64(rads)), math.Cos(float64(rads))))
	}

	return rads
}

func (t *Task) SetHeading(heading float32) {
	t.OnCourse = false
	t.heading = heading
	t.speed = 0
}

func (t *Task) DriveHeading(speed, heading float32) {
	t.OnCourse = false
	t.heading = heading
	t.speed = speed
}

func (t *Task) Tick(buttons input.ButtonState) {
	e := math.Pi / 150
	maxRot := float64(t.platform.GetMaxOmega()) * 0.5
	minRot := 0.01 * maxRot

	_, theta := t.model.GetPose()

	dTheta := float64(normalise(t.heading - theta))

	val := float64(dTheta) - math.Copysign(e, dTheta)
	if math.Signbit(val) != math.Signbit(dTheta) {
		val = 0
		t.OnCourse = true
	}
	val = val / (math.Pi - e)

	w := maxRot * val
	speed := float64(t.speed) - val * float64(t.speed)

	if math.Abs(w) < minRot {
		w = math.Copysign(minRot, w)
	}

	t.platform.SetArc(float32(speed), float32(w))
}

func NewTask(m *model.Model, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		model: m,
	}
}
