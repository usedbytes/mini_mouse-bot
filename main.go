// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package main

import (
	"image"
	"log"
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
)

type Pose struct {
	X, Y float64
	Heading float64
}

type Telem struct {
	lock sync.Mutex
	Euler []float64
	Pose Pose
	Frame image.Gray
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

func (t *Telem) GetFrame(ignored bool, img *image.Gray) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	*img = t.Frame

	return nil
}

func (t *Telem) SetFrame(img *image.Gray) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.Frame = *img
	t.Frame.Pix = make([]byte, len(img.Pix))
	copy(t.Frame.Pix, img.Pix)
}

func (t *Telem) GetPose(ignored bool, pose *Pose) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	*pose = t.Pose

	return nil
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

	lineTask := line.NewTask(platform)

	planner := plan.NewPlanner()
	planner.AddTask(line.TaskName, lineTask)
	planner.AddTask(waypoint.TaskName, wpTask)
	planner.AddTask(rc.TaskName, rc.NewTask(ip, platform))
	planner.SetTask(rc.TaskName)


	tick := time.NewTicker(16 * time.Millisecond)

	lastTime := time.Now()

	platform.Camera.Enable()
	defer platform.Camera.Disable()

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
			telem.SetFrame(&frame.Gray)
			lastTime = frameTime
		}

		buttons := ip.Buttons()
		if buttons[input.Triangle] == input.Pressed {
			mod.ResetOrientation()
		}

		if buttons[input.Square] == input.Pressed {
			log.Println("Square.")
			planner.SetTask("waypoint")
		}

		if buttons[input.Cross] == input.Pressed {
			planner.SetTask(line.TaskName)
		}

		if buttons[input.Circle] == input.Pressed {
			planner.SetTask("rc")
		}

		planner.Tick()
	}
}
