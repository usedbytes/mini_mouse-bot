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
}

func (t *Task) Enter() {
	t.platform.EnableCamera()
}

func (t *Task) Exit() {
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

	for i := frame.Bounds().Dy() - 7; i > 0; i-- {
		if !math.IsNaN(float64(line[i])) {
			v := float32(30.0 - math.Abs(float64(line[i])) * 50)
			w := 10 * line[i]
			fmt.Printf("Line at %+1.2f. v, w: %+2.2f, %+2.2f\n", line[i], v, w)
			t.platform.SetArc(v, w)
			break;
		}
	}
}

func NewTask(pl *base.Platform) *Task {
	return &Task{
		platform: pl,
	}
}
