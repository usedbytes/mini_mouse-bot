// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package bounce

import (
	"math"
	"time"

	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/model"
	"github.com/usedbytes/mini_mouse/bot/plan/heading"
	"github.com/usedbytes/mini_mouse/cv"
	"github.com/usedbytes/picamera"
)

const TaskName = "bounce"

type Task struct {
	platform *base.Platform
	heading *heading.Task
	running bool
	turning bool
	dir float32
	lastTime time.Time
}

func (t *Task) Enter() {
	t.platform.DisableCamera()
	t.platform.SetCameraCrop(picamera.Rect(0.0, 0.3, 1.0, 1.0))
	t.platform.SetCameraFormat(picamera.FORMAT_I420)
	t.platform.Camera.SetOutSize(32, 64)
	t.platform.EnableCamera()

	t.dir = 0.0
	t.running = false
	t.turning = false
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
		buttons[input.Cross] = input.None
		if t.running {
			t.platform.SetVelocity(0, 0)
		}
		t.running = !t.running
	}

	horz := cv.FindHorizon(frame)

	if !t.running {
		return
	}

	if t.turning {
		if !t.heading.OnCourse {
			t.heading.Tick(buttons)
			return
		}
		t.turning = false
	}

	if horz <= 0.40 {
		t.dir += float32(-math.Pi / 2)
		t.heading.SetHeading(t.dir)
		t.turning = true
	} else {
		if math.IsNaN(float64(horz)) {
			horz = 1.0
		}
		speed := t.platform.GetMaxVelocity() * horz
		t.heading.DriveHeading(speed, t.dir)
		t.heading.Tick(buttons)
		//t.platform.SetArc(t.platform.GetMaxVelocity() * horz * horz, 0)
	}
}

func NewTask(m *model.Model, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		heading: heading.NewTask(m, pl),
	}
}
