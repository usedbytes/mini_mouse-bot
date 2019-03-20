// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package base

import (
	"image/color"
	"math"
	"net"
	"log"
	"time"

	"periph.io/x/periph/host"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"github.com/usedbytes/bno055"
	"github.com/usedbytes/bot_matrix/datalink/netconn"
	"github.com/usedbytes/linux-led"
	"github.com/usedbytes/mini_mouse/bot/base/dev"
	"github.com/usedbytes/mini_mouse/bot/base/motor"
	"github.com/usedbytes/mini_mouse/bot/base/servo"
	"github.com/usedbytes/picamera"
)

type Boost int
const (
	BoostNone Boost = iota
	BoostSlow
	BoostFast
)

type Platform struct {
	dev *dev.Dev
	mmPerRev float32
	wheelbase float32

	Motors *motor.Motors
	boost Boost

	reconTime time.Duration

	lowBat gpio.PinIO

	i2cBus i2c.BusCloser
	imu *bno055.Dev
	vec []float64

	Camera *picamera.Camera
	frame picamera.Frame
	frameTime time.Time

	servos *servo.Dev
	reServos func() bool

	led led.RGBLED
	ledColor color.Color
	ledTrigger led.Trigger
}

func (p *Platform) AddLed(rgb led.RGBLED) {
	p.led = rgb

	p.SetLEDTrigger(p.ledTrigger)
	p.UpdateLed()
}

func (p *Platform) SetLEDTrigger(trig led.Trigger) {
	if p.led == nil {
		return
	}

	p.ledTrigger = trig
	p.led.SetTrigger(p.ledTrigger)
	p.UpdateLed()
}

func (p *Platform) SetLEDColor(c color.Color) {
	p.ledColor = c
	p.UpdateLed()
}

func (p *Platform) ResetLEDColor() {
	p.SetLEDTrigger(led.TriggerHeartbeat)
	p.ledColor = color.NRGBA{0x00, 0xff, 0x00, 0x80}
	p.UpdateLed()
}

func (p *Platform) UpdateLed() {
	if p.led == nil {
		return
	}

	p.led.SetColor(p.ledColor)
}

func (p *Platform) SetVelocity(a, b float32) {
	aRps := a / p.mmPerRev
	bRps := b / p.mmPerRev

	p.Motors.SetRPS(aRps, bRps)
}

func (p *Platform) SetOmega(w float32) {
	rps := w * (p.wheelbase / 2) / p.mmPerRev

	p.Motors.SetRPS(-rps, rps)
}

func (p *Platform) SetBoost(b Boost) {
	p.boost = b
}

func maxAbsAB(a, b float64) (abs float64, bNotA bool, sign float64) {
	if math.Abs(a) > math.Abs(b) {
		return math.Abs(a), false, math.Copysign(1.0, a)
	}

	return math.Abs(b), true, math.Copysign(1.0, b)
}

func (p *Platform) SetArc(vel, w float32) {
	platformMax := float64(p.GetMaxVelocity())
	deltaV := float64(w * p.wheelbase)

	if deltaV > 2 * platformMax {
		deltaV = 2 * platformMax
	}

	aVel := float64(vel) + deltaV / 2
	bVel := float64(vel) - deltaV / 2

	maxReq, bNotA, sign := maxAbsAB(float64(aVel), float64(bVel))
	if maxReq > platformMax {
		if bNotA {
			bVel = math.Copysign(platformMax, sign)
			aVel = bVel - math.Copysign(deltaV, sign)
		} else {
			aVel = math.Copysign(platformMax, sign)
			bVel = aVel - math.Copysign(deltaV, sign)
		}
	}

	aRps := float32(aVel) / p.mmPerRev
	bRps := float32(bVel) / p.mmPerRev

	p.Motors.SetRPS(aRps, bRps)
}

func (p *Platform) ControlledMove(mm, vel float32) {
	revs := float32(math.Abs(float64(mm))) / p.mmPerRev
	vel = vel / p.mmPerRev

	p.Motors.ControlledMove(revs, vel, revs, vel)
}

// FIXME: This doesn't really work, presumably due to wheel slippage
// This paper looks like it would be useful: http://www.ijmmm.org/papers/077-A009.pdf
func (p *Platform) ControlledArc(radius, tangential, angle float32) {
	innerR, outerR := radius - (p.wheelbase / 2), radius + (p.wheelbase / 2)
	innerMM, outerMM := innerR * angle, outerR * angle
	w := tangential / radius
	innerRPS, outerRPS := w * innerR / p.mmPerRev, w * outerR / p.mmPerRev

	innerRevs := float32(math.Abs(float64(innerMM))) / p.mmPerRev
	outerRevs := float32(math.Abs(float64(outerMM))) / p.mmPerRev

	if angle > 0 {
		p.Motors.ControlledMove(outerRevs, outerRPS, innerRevs, innerRPS)
	} else {
		p.Motors.ControlledMove(innerRevs, innerRPS, outerRevs, outerRPS)
	}
}

func (p *Platform) Moving() bool {
	return p.Motors.Moving()
}

func (p *Platform) GetMaxVelocity() float32 {
	max := p.Motors.GetMaxRPS()

	if p.boost == BoostSlow {
		max *= 0.3
	} else if p.boost == BoostFast {
		max *= 2
	}

	return max * p.mmPerRev
}

func (p *Platform) GetMaxOmega() float32 {
	deltaV := p.GetMaxVelocity()
	return deltaV * 2 / p.wheelbase
}

func (p *Platform) GetVelocity() (float32, float32) {
	a, b := p.Motors.GetRPS()
	return a * p.mmPerRev, b * p.mmPerRev
}

func (p *Platform) GetDistance() (float32, float32) {
	a, b := p.Motors.GetRevolutions()
	return a * p.mmPerRev, b * p.mmPerRev
}

func (p *Platform) Wheelbase() float32 {
	return p.wheelbase
}

func deg2rad(deg float32) float32 {
	return deg * math.Pi / 180.0
}

func (p *Platform) GetRot() float32 {
	if p.vec != nil {
		return deg2rad(float32(p.vec[0]))
	}
	return 0.0
}

func (p *Platform) SetCameraFormat(format picamera.Format) {
	p.Camera.SetFormat(format)
}

func (p *Platform) SetCameraCrop(crop picamera.Rectangle) {
	p.Camera.SetCrop(crop)
}

func (p *Platform) GetFrame() (picamera.Frame, time.Time) {
	return p.frame, p.frameTime
}

func (p *Platform) EnableCamera() {
	p.Camera.Enable()
}

func (p *Platform) DisableCamera() {
	if p.frame != nil {
		p.frame.Release()
		p.frame = nil
	}
	p.Camera.Disable()
}

func (p *Platform) CameraEnabled() bool {
	return p.Camera.Enabled()
}

func (p *Platform) Reconnect(recon func() bool) {
	time.AfterFunc(p.reconTime, func() {
		if !recon() {
			p.Reconnect(recon)
		}
	})
}

// FIXME: Broken encapsulation for music hacks.
func (p *Platform) Dev() *dev.Dev {
	return p.dev
}

func NewPlatform(/* Some config */) (*Platform, error) {
	_, err := host.Init()
	if err != nil {
		log.Fatal(err)
	}

	b, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}

	g := gpioreg.ByName("GPIO27")
	if g == nil {
		log.Fatal("Couldn't get low battery GPIO")
	}

	if err = g.In(gpio.PullDown, gpio.NoEdge); err != nil {
		log.Fatal(err)
	}

	c, err := net.Dial("unix", "/tmp/sock")
	if err != nil {
		log.Fatal(err)
	}
	t := netconn.NewNetconn(c)
	dev := dev.NewDev(t)

	p := &Platform{
		dev: dev,
		mmPerRev: (30.5 * math.Pi),
		wheelbase: 76,
		i2cBus: b,
		lowBat: g,
		reconTime: time.Second * 5,
		ledColor: color.NRGBA{0x00, 0xff, 0x00, 0x80},
		ledTrigger: led.TriggerHeartbeat,
	}

	p.Motors = motor.NewMotors(dev)

	p.Camera = picamera.NewCamera(64, 64, 60)
	if p.Camera == nil {
		log.Fatal("Couldn't open camera")
	}
	p.Camera.SetTransform(0, true, true)
	// FIXME: Should be configurable
	p.Camera.SetCrop(picamera.Rect(0, 0.5, 1.0, 1.0))

	imu, err := bno055.NewI2C(b, 0x29)
	if err != nil {
		log.Println("Couldn't get BNO055")
	} else {
		p.imu = imu
		err = p.imu.SetUseExternalCrystal(true)
		if err != nil {
			log.Println("IMU: SetUseExternalCrystal failed")
		}
	}

	p.reServos = func() bool {
		servos, err := servo.NewI2C(b, 0x40)
		if err != nil {
			log.Println("Couldn't get Servos")
			return false
		}

		p.servos = servos
		p.servos.SetTimeout(time.Second * 10)
		p.servos.SetPos(0.0, 0.0)
		p.servos.Enable(true, true)

		return true
	}

	haveServos := p.reServos()
	if !haveServos {
		p.Reconnect(p.reServos)
	}

	return p, nil
}

func (p *Platform) EnableServos(a, b bool) error {
	if p.servos == nil {
		return nil
	}

	err := p.servos.Enable(a, b)
	if err != nil {
		p.servos = nil
		p.Reconnect(p.reServos)
	}

	return err
}

func (p *Platform) SetServos(a, b float32) error {
	if p.servos == nil {
		return nil
	}

	err := p.servos.SetPos(a, b)
	if err != nil {
		p.servos = nil
		p.Reconnect(p.reServos)
	}

	return err
}

func (p *Platform) Update() error {
	pkts, err := p.dev.Poll()
	if err != nil {
		return err
	}

	low := p.lowBat.Read()
	if low == gpio.High {
		p.ledColor = color.NRGBA{0xff, 0x00, 0x00, 0x80}
		p.UpdateLed()
	}

	if p.Camera != nil {
		frame, _ := p.Camera.GetFrame(0)
		if frame != nil {
			if p.frame != nil {
				p.frame.Release()
			}
			p.frame = frame
			p.frameTime = time.Now()
		}
	}

	for _, pkt := range pkts {
		switch t := pkt.(type) {
		case (*motor.StepReport):
			p.Motors.AddSteps(t)
		default:
			if pkt != nil {
				log.Printf("%v\n", pkt)
			}
		}
	}

	if p.imu != nil {
		vec, err := p.imu.GetVector(bno055.VECTOR_EULER)
		if err != nil {
			log.Println("IMU: GetVector failed", err)
		}

		p.vec = vec
	}

	return nil
}
