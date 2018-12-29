// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package main

import (
	"encoding/gob"
	"image"
	"log"
	"math"
	"net"
	"net/rpc"
	"net/http"
	"sync"
	"time"

	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/model"
	"github.com/usedbytes/mini_mouse/bot/plan"
	"github.com/usedbytes/mini_mouse/bot/plan/rc"
	"github.com/usedbytes/mini_mouse/bot/plan/line"
	"github.com/usedbytes/mini_mouse/bot/plan/waypoint"
	"github.com/usedbytes/mini_mouse/bot/plan/heading"
	"github.com/usedbytes/picamera"
)

type Pose struct {
	X, Y float64
	Heading float64
}

type Telem struct {
	lock sync.Mutex
	Euler []float64
	Pose Pose
	Frame image.Image
}

func (t *Telem) SetEuler(vec []float64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.Euler = vec
}

func (t *Telem) GetEuler(ignored bool, vec *[]float64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	*vec = t.Euler

	return nil
}

func (t *Telem) SetPose(x, y, heading float64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.Pose = Pose{X: x, Y: y, Heading: heading}
}

func (t *Telem) GetFrame(ignored bool, img *image.Image) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	*img = t.Frame

	return nil
}

func (t *Telem) SetFrame(img image.Image) {
	t.lock.Lock()
	defer t.lock.Unlock()

	switch v := img.(type) {
	case *image.Gray:
	case *picamera.GrayFrame:
		pix := make([]byte, len(v.Pix))
		copy(pix, v.Pix)
		t.Frame = &image.Gray{
			Stride: v.Stride,
			Rect: v.Rect,
			Pix: pix,
		}
	case *image.YCbCr:
	case *picamera.YCbCrFrame:
		y := make([]byte, len(v.Y))
		copy(y, v.Y)
		cb := make([]byte, len(v.Cb))
		copy(cb, v.Cb)
		cr := make([]byte, len(v.Cr))
		copy(cr, v.Cr)
		t.Frame = &image.YCbCr{
			Y: y,
			Cb: cb,
			Cr: cr,
			YStride: v.YStride,
			CStride: v.CStride,
			SubsampleRatio: v.SubsampleRatio,
			Rect: v.Rect,
		}
	case *image.NRGBA:
	case *picamera.RGBFrame:
		pix := make([]byte, len(v.Pix))
		copy(pix, v.Pix)
		t.Frame = &image.NRGBA{
			Stride: v.Stride,
			Rect: v.Rect,
			Pix: pix,
		}
	default:
		log.Printf("%+v\n", v)
		panic("bad image type")
	}
}

func (t *Telem) GetPose(ignored bool, pose *Pose) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	*pose = t.Pose

	return nil
}

func init() {
	gob.Register(&image.NRGBA{})
	gob.Register(&image.Gray{})
	gob.Register(&image.YCbCr{})
}

func main() {
	log.Println("Mini Mouse")

	ip := input.NewCollector()

	telem := Telem{Euler: make([]float64, 3)}

	rpc.Register(&telem)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}
	go http.Serve(l, nil)

	platform, err := base.NewPlatform()
	if (err != nil) {
		log.Fatalf(err.Error())
	}
	mod := model.NewModel(platform)

	wpTask := waypoint.NewTask(mod, platform)
	wpTask.SetWaypoint(model.Coord{ 0, 0 })

	headingTask := heading.NewTask(mod, platform)
	headingTask.SetHeading(0.0)

	lineTask := line.NewTask(platform)

	planner := plan.NewPlanner()
	planner.AddTask(line.TaskName, lineTask)
	planner.AddTask(waypoint.TaskName, wpTask)
	planner.AddTask(heading.TaskName, headingTask)
	planner.AddTask(rc.TaskName, rc.NewTask(ip, platform))
	planner.SetTask(rc.TaskName)


	tick := time.NewTicker(16 * time.Millisecond)

	lastTime := time.Now()
	tmpTime := time.Now()
	dir := float32(0.0)

	for _ = range tick.C {
		err = platform.Update()
		if err != nil {
			log.Println(err.Error())
		}
		mod.Tick()

		pos, angle := mod.GetPose()
		telem.SetPose(float64(pos.X), float64(pos.Y), float64(angle))

		frame, frameTime := platform.GetFrame()
		if frame != nil && frameTime != lastTime {
			telem.SetFrame(frame)
			lastTime = frameTime
		}

		buttons := ip.Buttons()
		planner.Tick(buttons)

		if buttons[input.Triangle] == input.Pressed {
			mod.ResetOrientation()
		}

		if buttons[input.Share] == input.Pressed {
			if platform.CameraEnabled() {
				platform.DisableCamera()
			} else {
				platform.SetCameraFormat(picamera.FORMAT_RGBA)
				platform.Camera.SetCrop(picamera.Rect(0.0, 0.0, 1.0, 1.0))
				platform.Camera.SetOutSize(160, 160)
				platform.EnableCamera()
			}
		}

		if buttons[input.Square] == input.Pressed {
			planner.SetTask(heading.TaskName)
			log.Println("Square.")
			dir = 0.0
			headingTask.DriveHeading(200, dir)
			tmpTime = time.Now()
		}

		if planner.CurrentTask() == headingTask && time.Since(tmpTime) >= 4 * time.Second {
			dir += math.Pi / 2
			headingTask.DriveHeading(200, dir)
			tmpTime = time.Now()
		}

		if buttons[input.Cross] == input.Pressed {
			planner.SetTask(line.TaskName)
		}

		if buttons[input.Circle] == input.Pressed {
			planner.SetTask("rc")
		}

		if buttons[input.L1] == input.Pressed {
			log.Println("L1.")
			headingTask.SetHeading(-math.Pi / 2)
			planner.SetTask(heading.TaskName)
		}

		if buttons[input.R1] == input.Pressed {
			log.Println("R1.")
			headingTask.SetHeading(math.Pi / 2)
			planner.SetTask(heading.TaskName)
		}
	}
}
