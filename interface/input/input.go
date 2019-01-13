package input

import (
	"image/color"
	"log"
	"time"

	"github.com/gvalkov/golang-evdev"
	"github.com/usedbytes/input2"
	"github.com/usedbytes/input2/button"
	"github.com/usedbytes/input2/gamepad/thumbstick"
	"github.com/usedbytes/input2/factory"

	"github.com/usedbytes/linux-led"
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
	L2
	L3
	R1
	R2
	R3
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
		}
	}
}

func (c *Collector) GetSticks() (float32, float32) {
	return c.leftStick, c.rightStick
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

func NewCollector() *Collector {
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
				rgbled.SetColor(color.NRGBA{0x00, 0xff, 0x00, 0xff})
				rgbled.SetTrigger(led.TriggerHeartbeat)
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
				{ evdev.BTN_TL2, L2 },
				{ evdev.BTN_THUMBL, L3 },
				{ evdev.BTN_TR, R1 },
				{ evdev.BTN_TR2, R2 },
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

			sub := conn.Subscribe(stopChan)
			go c.handleEvents(sub)
		}
	}()

	return c
}
