// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package waypoint

import (
	"image/color"
	"log"
	"math"

	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/model"
)

const TaskName = "waypoint"

type Task struct {
	platform *base.Platform
	model *model.Model

	waypoint model.Coord
}

func (t *Task) SetWaypoint(c model.Coord) {
	t.waypoint = c
}

func (t *Task) Tick(buttons input.ButtonState) {
	pos, theta := t.model.GetPose()

	dPos := t.waypoint.Sub(pos)
	heading := float32(math.Atan2(float64(dPos.Y), float64(dPos.X)))
	dTheta := heading - theta
	hypot := math.Hypot(float64(dPos.X), float64(dPos.Y))
	if hypot <= 30 {
		log.Printf("Arrived\n")
		t.platform.SetVelocity(0, 0)
		return
	} else if math.Abs(float64(dTheta)) > (math.Pi / 25)  {
		// Rotate
		//w := math.Max(math.Pi / 16, math.Min(math.Abs(float64(dTheta)), math.Pi / 2))
		w := 1.0
		if dTheta < 0 {
			w = -w
		}
		t.platform.SetOmega(float32(w))
	} else  {
		// Move
		maxSpeed := t.platform.GetMaxVelocity()
		v := math.Min(math.Abs(float64(hypot)), float64(maxSpeed * 0.75))
		t.platform.SetVelocity(float32(v), float32(v))
	}

	log.Printf("dPos: %v, heading: %v dtheta: %v\n", dPos, heading * 180 / math.Pi, dTheta * 180 / math.Pi)
}

func (t *Task) Color() color.Color {
	return color.NRGBA{ 0xf4, 0x9e, 0x42, 0x80 }
}

func NewTask(m *model.Model, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		model: m,
	}
}
