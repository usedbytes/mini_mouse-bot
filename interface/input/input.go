package input

import (
	"log"
	"time"

	"github.com/gvalkov/golang-evdev"
	"github.com/usedbytes/input2"
	"github.com/usedbytes/input2/button"
	"github.com/usedbytes/input2/gamepad/hat"
	"github.com/usedbytes/input2/gamepad/thumbstick"
	"github.com/usedbytes/input2/gamepad/trigger"
	"github.com/usedbytes/input2/factory"

	"github.com/usedbytes/linux-led"

	"github.com/usedbytes/mini_mouse/bot/base"
)

type Button int
const (
	Cross Button = iota
	Square
	Triangle
	Circle
	PS
	Share
	Options
	L1
	L3
	R1
	R3
	Up = evdev.KEY_UP
	Down = evdev.KEY_DOWN
	Left = evdev.KEY_LEFT
	Right = evdev.KEY_RIGHT
)

type State int
const (
	None State = iota
	Pressed
	Held
)

type ButtonState map[Button]State

type Collector struct {
	leftStick, rightStick float32
	leftTrigger, rightTrigger float32
	buttons ButtonState
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
		case trigger.Event:
			if e.Code == trigger.Left {
				c.leftTrigger = e.Value
			} else {
				c.rightTrigger = e.Value
			}
		}
	}
}

func (c *Collector) GetSticks() (float32, float32) {
	return c.leftStick, c.rightStick
}

func (c *Collector) GetTriggers() (float32, float32) {
	return c.leftTrigger, c.rightTrigger
}

func (c *Collector) Buttons() ButtonState {
	defer func() {
		c.buttons = make(ButtonState)
	}()
	return c.buttons
}

type buttonMap struct {
	scancode uint16
	button Button
}

func NewCollector(p *base.Platform) *Collector {
	c := &Collector{
		buttons: make(ButtonState),
	}

	stopChan := make(chan bool)

	go func() {
		sources := factory.Monitor()
		for s := range sources {
			log.Println("Source: ", s)
			conn := s.NewConnection()

			rgbled, ok := s.(led.RGBLED)
			if ok {
				p.AddLed(rgbled)
			}

			btnMap := []buttonMap{
				{ evdev.BTN_MODE, PS },
				{ evdev.BTN_NORTH, Triangle },
				{ evdev.BTN_EAST, Circle },
				{ evdev.BTN_SOUTH, Cross },
				{ evdev.BTN_WEST, Square },
				{ evdev.BTN_SELECT, Share },
				{ evdev.BTN_START, Options },
				{ evdev.BTN_TL, L1 },
				{ evdev.BTN_THUMBL, L3 },
				{ evdev.BTN_TR, R1 },
				{ evdev.BTN_THUMBR, R3 },
			}

			for _, b := range btnMap {
				button.MapButton(conn,
					&button.Button{
						Match: input2.EventMatch{evdev.EV_KEY, b.scancode},
						HoldTime: (time.Millisecond * 1500),
						Keycode: int(b.button),
					})
			}

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

			hat.MapHat(conn,
				&hat.Hat{
					X: hat.Axis{ Code: evdev.ABS_HAT0X, HoldTime: time.Millisecond * 800 },
					Y: hat.Axis{ Code: evdev.ABS_HAT0Y, HoldTime: time.Millisecond * 800, Invert: true },
				},
			)

			trigger.MapTrigger(conn,
				&trigger.Trigger{ Axis: evdev.ABS_Z, Code: trigger.Left })
			trigger.MapTrigger(conn,
				&trigger.Trigger{ Axis: evdev.ABS_RZ, Code: trigger.Right })

			sub := conn.Subscribe(stopChan)
			go c.handleEvents(sub)
		}
	}()

	return c
}
