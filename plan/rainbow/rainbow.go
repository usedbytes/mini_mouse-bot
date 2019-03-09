// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package rainbow

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/model"
	"github.com/usedbytes/mini_mouse/bot/plan/heading"
	"github.com/usedbytes/mini_mouse/cv"
	"github.com/usedbytes/picamera"
)

type State int
const (
	Pirouette State = iota
	Confirm
	GoTo
	Approach
)

type Corner struct {
	dir float32
	coords image.Point
	c color.Color
}

var coords []image.Point = []image.Point{
	{  1,  1 },
	{  1, -1 },
	{ -1, -1 },
	{ -1,  1 },
}

func NewCorner(idx int, c color.Color) Corner {
	return Corner {
		dir: float32(2 * idx + 1) * math.Pi / 4,
		coords: coords[idx],
		c: c,
	}
}

func (c Corner) GoTo(to Corner) (float32, float32) {
	fromCoords, toCoords := c.coords, to.coords
	side := 900.0
	//side := 10.0

	dx := float64(toCoords.X - fromCoords.X) * side / 2
	dy := float64(toCoords.Y - fromCoords.Y) * side / 2
	heading := float64(math.Atan2(dx, dy))
	distance := math.Sqrt(dx * dx + dy * dy)

	return float32(heading), float32(distance)
}

const TaskName = "rainbow"

type Task struct {
	platform *base.Platform
	mod *model.Model
	heading *heading.Task
	running bool
	turning bool
	moving bool
	fancy bool
	dir float32
	dist float32
	lastTime time.Time

	colors []color.Color
	state State
	corners []Corner
	corner int
	subState int
}

func (t *Task) reset() {
	t.corners = t.corners[:0]
	t.colors = make([]color.Color, 0, 4)

	t.dir = 0.0
	t.running = false
	t.turning = false
	t.fancy = false

	t.state = Pirouette

	t.mod.ResetOrientation()
	t.platform.SetLEDColor(t.Color())

	rand.Seed(time.Now().UnixNano())
}

func (t *Task) Enter() {
	t.platform.DisableCamera()
	t.platform.SetCameraCrop(picamera.Rect(0.0, 0.0, 1.0, 1.0))
	t.platform.SetCameraFormat(picamera.FORMAT_I420)
	t.platform.Camera.SetOutSize(100, 100)
	t.platform.EnableCamera()

	t.platform.SetLEDColor(t.Color())
	t.running = false

	if t.colors == nil {
		t.reset()
	}
}

func (t *Task) Exit() {
	t.platform.SetVelocity(0, 0)
	t.platform.DisableCamera()
}

func (t *Task) tickPirouette(img image.Image) {
	w, h := cv.ImageDims(img)

	// Keep turning
	if t.turning {
		if !t.heading.OnCourse {
			t.heading.Tick(nil)
			return
		}
		t.turning = false
	}

	// Initial turn from start forwards
	if len(t.colors) == 0 && t.dir == 0.0 {
		t.dir = float32(math.Pi / 4)
		t.heading.SetHeading(t.dir)
		t.turning = true
		t.heading.Tick(nil)
		return
	}

	// Reached one
	stripeH := h / 8
	roi := image.Rect(0, (h / 2) - (stripeH / 2), w, (h / 2) + (stripeH / 2))
	left, right, _ := cv.FindBoard(img, nil, roi)
	c := img.At(int(((left + right) / 2) * float32(w)), h / 4)
	t.colors = append(t.colors, c)

	if len(t.colors) == 4 {
		t.state = GoTo
		t.corner = 0
		t.subState = 0

		maxCr := cv.Tuple{ -1, 0 }
		maxCb := cv.Tuple{ -1, 0 }
		minArg := cv.Tuple{ -1, 9999 }
		minCb := cv.Tuple{ -1, 9999 }
		for i, c := range t.colors {
			v := c.(color.YCbCr)

			if int(v.Cr) > maxCr.Second {
				maxCr.First = i
				maxCr.Second = int(v.Cr)
			}

			if int(v.Cb) > maxCb.Second {
				maxCb.First = i
				maxCb.Second = int(v.Cb)
			}

			x := float64(int(v.Cb) - 128)
			y := float64(int(v.Cr) - 128)
			arg := int(math.Sqrt(x * x + y * y))
			if arg < minArg.Second {
				minArg.First = i
				minArg.Second = arg
			}

			if int(v.Cb) < minCb.Second {
				minCb.First = i
				minCb.Second = int(v.Cb)
			}
		}

		ordered := make([]Corner, 4)
		ordered[0] = NewCorner(maxCr.First, t.colors[maxCr.First])
		ordered[1] = NewCorner(maxCb.First, t.colors[maxCb.First])
		ordered[2] = NewCorner(minCb.First, t.colors[minCb.First])
		ordered[3] = NewCorner(minArg.First, t.colors[minArg.First])

		//rand.Shuffle(len(ordered), func(i, j int) { ordered[i], ordered[j] = ordered[j], ordered[i] })
		t.corners = ordered

		t.platform.SetLEDColor(t.Color())

		return
	}

	t.dir += float32(math.Pi / 2)
	t.heading.SetHeading(t.dir)
	t.turning = true
	t.heading.Tick(nil)

	return
}

func (t *Task) tickConfirm(img image.Image) {
	// Keep turning
	if t.turning {
		if !t.heading.OnCourse {
			t.heading.Tick(nil)
			return
		}
		t.turning = false
	}

	if t.corner == -1 {
		t.corner = 0
	} else {
		left, right, bottom := cv.FindBoard(img, t.corners[t.corner].c, img.Bounds())

		fmt.Println("Board", t.corner, left, right, bottom)

		t.corner++
		if t.corner >= 4 {
			t.state = Approach
			return
		}
	}

	t.dir = t.corners[t.corner].dir
	t.heading.SetHeading(t.dir)
	t.turning = true
	t.heading.Tick(nil)

	return
}

func minimiseDir(current, next, dist float32) (float32, float32) {
	dtheta := next - current
	dtheta = heading.Normalise(dtheta)
	if dtheta < -math.Pi / 2 {
		return heading.Normalise(next + math.Pi), -dist
	}

	if dtheta > math.Pi / 2 {
		return heading.Normalise(next - math.Pi), -dist
	}

	return next, dist
}

func (t *Task) tickGoToSimple(img image.Image) {
	// Keep turning
	if t.turning {
		if !t.heading.OnCourse {
			t.heading.Tick(nil)
			return
		}
		t.turning = false
	}

	// Keep moving
	if t.moving {
		if t.platform.Moving() {
			return
		}
		t.platform.SetVelocity(0, 0)
		t.moving = false
		return
	}

	switch t.subState {
	case 0:
		t.dir = t.corners[t.corner].dir
		t.platform.SetBoost(base.BoostNone)
		t.heading.SetHeading(t.dir)
		t.turning = true
		t.heading.Tick(nil)
		t.subState = 1
	case 1:
		t.platform.SetBoost(base.BoostFast)
		t.platform.ControlledMove(610, t.platform.GetMaxVelocity())
		t.moving = true
		t.subState = 2
	case 2:
		if t.corner < 3 {
			t.platform.SetBoost(base.BoostFast)
			t.platform.ControlledMove(610, -t.platform.GetMaxVelocity())
			t.moving = true
		}
		t.subState = 3
	case 3:
		t.corner++
		t.subState = 0
		if t.corner >= 4 {
			t.running = false
			t.state = GoTo
			t.corner = 0
		}
	}

	return
}

func (t *Task) tickGoToFancy(img image.Image) {
	// Keep turning
	if t.turning {
		if !t.heading.OnCourse {
			t.heading.Tick(nil)
			return
		}
		t.turning = false
	}

	// Keep moving
	if t.moving {
		if t.platform.Moving() {
			return
		}
		t.platform.SetVelocity(0, 0)
		t.moving = false
		return
	}

	switch t.subState {
	case 0:
		if t.corner == 0 {
			t.dir = t.corners[0].dir
			t.dist = 610
			//t.dist = 5
		} else {
			dir, dist := t.corners[t.corner - 1].GoTo(t.corners[t.corner])
			t.dir, t.dist = minimiseDir(t.dir, dir, dist)
		}
		t.platform.SetBoost(base.BoostNone)
		t.heading.SetHeading(t.dir)
		t.turning = true
		t.heading.Tick(nil)
		t.subState = 1
	case 1:
		t.platform.SetBoost(base.BoostFast)
		t.platform.ControlledMove(t.dist, float32(math.Copysign(float64(t.platform.GetMaxVelocity()), float64(t.dist))))
		t.moving = true
		t.subState = 3
	case 2:
		if t.corner < 3 {
			t.platform.SetBoost(base.BoostFast)
			t.platform.ControlledMove(610, -t.platform.GetMaxVelocity())
			t.moving = true
		}
		t.subState = 3
	case 3:
		t.corner++
		t.subState = 0
		if t.corner >= 4 {
			t.running = false
			t.state = GoTo
			t.corner = 0
		}
	}

	return
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

	if buttons[input.Options] == input.Pressed {
		buttons[input.Options] = input.None
		t.fancy = !t.fancy
		t.platform.SetLEDColor(t.Color())
	}

	if buttons[input.Square] == input.Pressed {
		buttons[input.Square] = input.None
		t.reset()
	}

	if !t.running {
		return
	}

	var img image.Image
	switch v := frame.(type) {
	case *picamera.YCbCrFrame:
		img = &v.YCbCr
	default:
		img = frame
	}

	switch t.state {
	case Pirouette:
		t.tickPirouette(img)
	case Confirm:
		t.tickConfirm(img)
	case GoTo:
		if t.fancy {
			t.tickGoToFancy(img)
		} else {
			t.tickGoToSimple(img)
		}
	}
}

func (t *Task) Color() color.Color {
	red := uint8(0)
	if t.fancy {
		red = 0x40
	}
	if t.corners != nil && len(t.corners) == 4 {
		return color.NRGBA{ red, 0xff, 0xff, 0x80 }
	}
	return color.NRGBA{ red, 0x40, 0xa0, 0x80 }
}

func NewTask(m *model.Model, pl *base.Platform) *Task {
	return &Task{
		platform: pl,
		mod: m,
		heading: heading.NewTask(m, pl),
		colors: make([]color.Color, 0, 4),
	}
}
