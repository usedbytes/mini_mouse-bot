package input

import (
	"log"
	"time"

	"github.com/gvalkov/golang-evdev"
	"github.com/usedbytes/input2"
	"github.com/usedbytes/input2/button"
	"github.com/usedbytes/input2/gamepad/thumbstick"
	"github.com/usedbytes/input2/factory"
)

type Button int
const (
	Cross Button = iota
	Square
	Triangle
	Circle
	PS
)

type State int
const (
	None State = iota
	Pressed
	Held
)

type Collector struct {
	leftStick, rightStick float32
	buttons map[Button]State
	held map[Button]bool
}

func (c *Collector) handleEvents(ch <-chan input2.InputEvent) {
	for ev := range ch {
		switch e := ev.(type) {
		case thumbstick.Event:
			mag := float32(e.Arg)

			if e.Stick == 0 {
				if (e.Theta > 90) && (e.Theta < 270) {
					mag = -mag
				}
				c.leftStick = mag
			} else {
				if (e.Theta > 180) {
					mag = -mag
				}
				c.rightStick = mag
			}

		case button.Event:
			if e.Value == button.Pressed {
				c.buttons[Button(e.Keycode)] = Pressed
			} else if e.Value == button.Held {
				c.buttons[Button(e.Keycode)] = Held
			}
		}
	}
}

func (c *Collector) GetSticks() (float32, float32) {
	return c.leftStick, c.rightStick
}

func (c *Collector) Buttons() map[Button]State {
	defer func() {
		c.buttons = make(map[Button]State)
	}()
	return c.buttons
}

func NewCollector() *Collector {
	c := &Collector{
		buttons: make(map[Button]State),
	}

	stopChan := make(chan bool)

	go func() {
		sources := factory.Monitor()
		for s := range sources {
			log.Println("Source: ", s)
			conn := s.NewConnection()

			button.MapButton(conn,
				&button.Button{
					Match: input2.EventMatch{evdev.EV_KEY, evdev.BTN_MODE},
					HoldTime: (time.Millisecond * 3000),
					Keycode: int(PS),
				})
			button.MapButton(conn,
				&button.Button{
					Match: input2.EventMatch{evdev.EV_KEY, evdev.BTN_NORTH},
					HoldTime: (time.Millisecond * 1500),
					Keycode: int(Triangle),
				})
			button.MapButton(conn,
				&button.Button{
					Match: input2.EventMatch{evdev.EV_KEY, evdev.BTN_EAST},
					HoldTime: (time.Millisecond * 1500),
					Keycode: int(Circle),
				})
			button.MapButton(conn,
				&button.Button{
					Match: input2.EventMatch{evdev.EV_KEY, evdev.BTN_SOUTH},
					HoldTime: (time.Millisecond * 1500),
					Keycode: int(Cross),
				})
			button.MapButton(conn,
				&button.Button{
					Match: input2.EventMatch{evdev.EV_KEY, evdev.BTN_WEST},
					HoldTime: (time.Millisecond * 1500),
					Keycode: int(Square),
				})
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
