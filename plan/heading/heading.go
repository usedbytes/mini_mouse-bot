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
	maxRot, minRot float64
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
	coarse := math.Pi / 150
	fine := math.Pi / 180

	_, theta := t.model.GetPose()

	dTheta := float64(normalise(t.heading - theta))

	val := 0.0
	if math.Abs(dTheta) <= fine {
		t.OnCourse = true
		t.platform.SetArc(float32(t.speed), float32(0))
		return
	} else if math.Abs(dTheta) > coarse {
		val = math.Max(math.Min((dTheta - math.Copysign(coarse, dTheta)) / (math.Pi / 2), 1.0), -1.0)
	}

	w := t.maxRot * val
	speed := float64(t.speed) - val * float64(t.speed)

	if math.Abs(w) < t.minRot {
		w = math.Copysign(t.minRot, w)
	}

	t.platform.SetArc(float32(speed), float32(w))
}

func NewTask(m *model.Model, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		model: m,
		maxRot: float64(pl.GetMaxOmega()),
		minRot: 0.3,
	}
}
