// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package line

import (
	"fmt"
	"math"
	"time"

	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/plan/line/algo"
)

const TaskName = "line"

type Task struct {
	platform *base.Platform

	lastTime time.Time
	running bool
	side float32
	lost, search int
	maxSpeed, maxTurn float32
}

func (t *Task) Enter() {
	t.platform.EnableCamera()
}

func (t *Task) Exit() {
	t.platform.SetVelocity(0, 0)
	t.platform.DisableCamera()
}

func (t *Task) Tick(buttons input.ButtonState) {
	frame, frameTime := t.platform.GetFrame()
	if frame == nil || frameTime == t.lastTime {
		return
	}
	t.lastTime = frameTime

	if buttons[input.Cross] == input.Pressed {
		if t.running {
			t.platform.SetVelocity(0, 0)
		}
		t.running = !t.running
	}

	if !t.running {
		return
	}

	line := algo.FindLine(&frame.Gray)

	h := frame.Bounds().Dy()
	nearest := h + 1
	furthest := -1

	for i, v := range line {
		if math.IsNaN(float64(v)) {
			continue
		}
		if i < nearest {
			nearest = i
		}
		if i > furthest {
			furthest = i
		}
	}

	if (nearest > h || furthest < 0) || (t.lost > 0 && nearest > h / 2) {
		fmt.Printf("Lost line! prev was %v\n", t.side)
		t.lost++
		if t.lost > t.search {
			t.side = -t.side
			t.search *= 2
		}
		t.platform.SetArc(0, float32(math.Copysign(5.0, float64(t.side))))
		return
	} else {
		t.lost = 0
		t.search = 60
	}

	val := float32(0.0)
	mid := int(math.Ceil(float64(furthest - nearest) / 2))
	for i := mid; i < furthest; i++ {
		if !math.IsNaN(float64(line[i])) {
			val = line[i]
			break
		}
	}

	if val > 0 || val < 0 {
		t.side = val
	}

	vel := float32(float64(t.maxSpeed) - math.Abs(float64(val)) * float64(2 * t.maxSpeed))
	omega := t.maxTurn * val
	t.platform.SetArc(vel, omega)
}

func NewTask(pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		search: 60,
		maxSpeed: 300,
		maxTurn: 10,
	}
}
