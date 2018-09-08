package input

import (
	"log"

	"github.com/gvalkov/golang-evdev"
	"github.com/usedbytes/input2"
	"github.com/usedbytes/input2/gamepad/thumbstick"
	"github.com/usedbytes/input2/factory"
)

type Collector struct {
	leftStick, rightStick float32
}

func (c *Collector) handleEvents(ch <-chan input2.InputEvent) {
	for ev := range ch {
		switch e := ev.(type) {
		case thumbstick.Event:
			mag := float32(e.Arg)
			if (e.Theta > 90) && (e.Theta < 270) {
				mag = -mag
			}

			if e.Stick == 0 {
				c.leftStick = mag
			} else {
				c.rightStick = mag
			}
		}
	}
}

func (c *Collector) GetSticks() (float32, float32) {
	return c.leftStick, c.rightStick
}

func NewCollector() *Collector {
	c := &Collector{

	}

	stopChan := make(chan bool)

	go func() {
		sources := factory.Monitor()
		for s := range sources {
			log.Println("Source: ", s)
			conn := s.NewConnection()

			thumbstick.MapThumbstick(conn,
				&thumbstick.Thumbstick{
					X: thumbstick.Axis{ Code: evdev.ABS_X },
					Y: thumbstick.Axis{ Code: evdev.ABS_Y, Invert: true },
					Stick: thumbstick.Left,
					Algo: thumbstick.CrossDeadzone{ XDeadzone: 0.2, YDeadzone: 0.2 },
				})
			thumbstick.MapThumbstick(conn,
				&thumbstick.Thumbstick{
					X: thumbstick.Axis{ Code: evdev.ABS_RX },
					Y: thumbstick.Axis{ Code: evdev.ABS_RY, Invert: true },
					Stick: thumbstick.Right,
					Algo: thumbstick.CrossDeadzone{ XDeadzone: 0.2, YDeadzone: 0.2 },
				})

			sub := conn.Subscribe(stopChan)
			go c.handleEvents(sub)
		}
	}()

	return c
}
