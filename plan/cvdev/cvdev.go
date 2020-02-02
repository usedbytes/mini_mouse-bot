// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package cvdev

import (
	"image"
	"image/color"
	"time"

	"github.com/usedbytes/thunk-bot/base"
	"github.com/usedbytes/thunk-bot/interface/input"
	"github.com/usedbytes/thunk-bot/plan"
	"github.com/usedbytes/thunk-bot/plan/rc"
	"github.com/usedbytes/mini_mouse/cv"
	"github.com/usedbytes/picamera"
)

const TaskName = "cvdev"

type Task struct {
	platform *base.Platform
	lastTime time.Time
	rcTask plan.Task
}

func (t *Task) Enter() {
	t.platform.DisableCamera()
	t.platform.SetCameraCrop(picamera.Rect(0.0, 0.0, 1.0, 1.0))
	t.platform.SetCameraFormat(picamera.FORMAT_I420)
	t.platform.Camera.SetOutSize(128, 128)
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

	t.rcTask.Tick(buttons)

	var img image.Image
	switch v := frame.(type) {
	case *picamera.YCbCrFrame:
		img = &v.YCbCr
	default:
		img = frame
	}

	cv.RunAlgorithm(img, nil, true)
}

func (t *Task) Color() color.Color {
	return color.NRGBA{ 0xff, 0xff, 0xff, 0x80 }
}

func NewTask(ip *input.Collector, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		rcTask: rc.NewTask(ip, pl),
	}
}

