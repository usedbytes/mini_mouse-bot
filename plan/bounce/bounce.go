// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package bounce

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/usedbytes/thunk-bot/base"
	"github.com/usedbytes/thunk-bot/interface/input"
	"github.com/usedbytes/thunk-bot/model"
	"github.com/usedbytes/thunk-bot/plan/heading"
	"github.com/usedbytes/mini_mouse/cv"
	"github.com/usedbytes/picamera"
)

const TaskName = "bounce"

const (
	Left = 0
	Right = 1
)

var maze []int = []int{ Left, Left, Right, Right, Left, Left, Left, Right }

type Task struct {
	platform *base.Platform
	mod *model.Model
	heading *heading.Task
	running bool
	turning bool
	dir float32
	lastTime time.Time

	max, min float32
	count int
	speedMultiplier float32
	turn int
}

func (t *Task) Enter() {
	t.platform.DisableCamera()
	t.platform.SetCameraCrop(picamera.Rect(0.0, 0.0, 1.0, 1.0))
	t.platform.SetCameraFormat(picamera.FORMAT_I420)
	t.platform.Camera.SetOutSize(40, 80)
	t.platform.EnableCamera()

	t.mod.ResetOrientation()

	t.dir = 0.0
	t.running = false
	t.turning = false

	t.turn = 0
}

func (t *Task) Exit() {
	t.platform.SetBoost(base.BoostNone)
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
		} else {
			if buttons[input.Up] == input.Held {
				buttons[input.Up] = input.None
				t.speedMultiplier = 2.0
			} else if buttons[input.Right] == input.Held {
				buttons[input.Right] = input.None
				t.speedMultiplier = 1.85
			} else if buttons[input.Left] == input.Held {
				buttons[input.Left] = input.None
				t.speedMultiplier = 1.7
			} else if buttons[input.Down] == input.Held {
				buttons[input.Down] = input.None
				t.speedMultiplier = 1.0
			} else {
				t.speedMultiplier = 1.5
			}
		}
		t.running = !t.running
	}

	var img image.Image
	switch v := frame.(type) {
	case *picamera.YCbCrFrame:
		img = &v.YCbCr
	default:
		img = frame
	}
	horz := 1.0 - cv.FindHorizon(img)
	//fmt.Println(horz)

	if !t.running {
		return
	}

	if t.turning {
		if !t.heading.OnCourse {
			t.heading.Tick(buttons)
			return
		}
		t.turning = false
		// Just enable us to reach higher speeds
		// The actual speed is determined by t.speedMultiplier
		t.platform.SetBoost(base.BoostFast)
	}

	if horz <= t.min {
		t.count++
		if t.count <= 5 {
			t.heading.DriveHeading(20, t.dir)
			t.heading.Tick(buttons)
		} else {
			if t.turn >= len(maze) {
				t.platform.SetVelocity(200, 200)
				t.running = false
				return
			}
			turn := maze[t.turn]

			if turn == Left {
				t.dir += float32(-math.Pi / 2)
			} else {
				t.dir += float32(math.Pi / 2)
			}
			t.platform.SetBoost(base.BoostNone)
			t.heading.SetHeading(t.dir)
			t.turning = true

			t.turn++
		}
	} else {
		t.count = 0
		if math.IsNaN(float64(horz)) || horz > t.max {
			horz = t.max
		}

		t.platform.SetBoost(base.BoostFast)
		slow := ((t.max - horz) / (t.max - t.min)) * 0.5

		maxSpeed := t.platform.GetMaxBoostedVelocity(base.BoostNone) * t.speedMultiplier
		speed := maxSpeed - (slow * maxSpeed)

		t.heading.DriveHeading(speed, t.dir)
		t.heading.Tick(buttons)
	}
}

func (t *Task) Color() color.Color {
	return color.NRGBA{ 0xff, 0x00, 0xff, 0x80 }
}

func NewTask(m *model.Model, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		mod: m,
		heading: heading.NewTask(m, pl),
		max: float32(0.50),
		min: float32(0.44),
		count: 0,
		speedMultiplier: float32(1.5),
	}
}
