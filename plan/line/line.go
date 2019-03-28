// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package line

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/plan/line/algo"
	"github.com/usedbytes/picamera"
)

const TaskName = "line"

type Task struct {
	platform *base.Platform

	lastTime time.Time
	running bool
	side float32
	lost, search int
	maxSpeed, maxTurn float32
	speedMultiplier float32
}

func (t *Task) Enter() {
	t.platform.DisableCamera()
	t.platform.SetCameraCrop(picamera.Rect(0.0, 0.6, 1.0, 1.0))
	t.platform.SetCameraFormat(picamera.FORMAT_YV12)
	t.platform.Camera.SetOutSize(32, 32)
	t.platform.EnableCamera()
	t.platform.SetBoost(base.BoostFast)

	t.running = false
}

func (t *Task) Exit() {
	t.platform.SetVelocity(0, 0)
	t.platform.DisableCamera()
	t.platform.SetBoost(base.BoostNone)
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

	grayFrame := frame.(*picamera.GrayFrame)
	line := algo.FindLine(&grayFrame.Gray)

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

	vel := float32(0)
	omega := float32(0)

	if (nearest > h || furthest < 0) || (t.lost > 0 && nearest > h / 2) || furthest == nearest {
		fmt.Printf("Lost line! prev was %v\n", t.side)
		t.lost++
		if t.lost > t.search {
			t.side = -t.side
			t.search *= 2
		}
		vel, omega = 0, float32(math.Copysign(float64(2.5), float64(t.side)))
	} else {
		t.lost = 0
		t.search = 60

		x1 := float32(furthest) / float32(h)
		y1 := line[furthest]

		x2 := float32(nearest) / float32(h)
		y2 := line[nearest]

		m := (y2 - y1) / (x2 - x1)
		c := y2 - m * x2

		//fmt.Printf("(%1.3f, %1.3f), (%2.3f, %1.3f) -> y = %2.3f * x + %2.3f\n", x1, y1, x2, y2, m, c);

		if c > 0 || c < 0 {
			t.side = c
		}

		if math.Abs(float64(m)) < 0.1 && math.Abs(float64(c)) < 0.1 {
			vel = t.platform.GetMaxBoostedVelocity(base.BoostNone) * t.speedMultiplier

			// Turn proportional to c, higher amplification than normal
			omega = t.maxTurn * c * 1.5
		} else {
			// Slow proportional to m
			reductor := float32(math.Min(0.95, math.Abs(float64(m))))
			vel = t.maxSpeed * (1 - reductor)

			// Turn proportional to c
			omega = t.maxTurn * c
		}

	}

	//fmt.Println(vel, omega)

	if !t.running {
		return
	}

	t.platform.SetArc(vel, omega)
}

func (t *Task) Color() color.Color {
	return color.NRGBA{ 0xff, 0xff, 0x00, 0x80 }
}

func NewTask(pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		search: 60,
		maxSpeed: 500,
		maxTurn: 7,
		speedMultiplier: float32(1.5),
	}
}
