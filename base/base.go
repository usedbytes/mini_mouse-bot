// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package base

import (
	"math"
	"net"
	"log"
	"time"

	"periph.io/x/periph/host"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"github.com/usedbytes/bno055"
	"github.com/usedbytes/bot_matrix/datalink/netconn"
	"github.com/usedbytes/mini_mouse/bot/base/dev"
	"github.com/usedbytes/mini_mouse/bot/base/motor"
	"github.com/usedbytes/picamera"
)

type Platform struct {
	dev *dev.Dev
	mmPerRev float32
	wheelbase float32

	Motors *motor.Motors

	i2cBus i2c.BusCloser
	imu *bno055.Dev
	vec []float64

	Camera *picamera.Camera
	frame picamera.Frame
	frameTime time.Time
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

func (p *Platform) SetArc(vel, w float32) {
	deltaV := (w * p.wheelbase) / 2

	aVel := vel + deltaV / 2
	bVel := vel - deltaV / 2

	red := float32(0.0)
	if aVel > p.GetMaxVelocity() {
		red = aVel - p.GetMaxVelocity()
	} else if aVel < -p.GetMaxVelocity() {
		red = aVel + p.GetMaxVelocity()
	} else if bVel > p.GetMaxVelocity() {
		red = bVel - p.GetMaxVelocity()
	} else if bVel < -p.GetMaxVelocity() {
		red = bVel + p.GetMaxVelocity()
	}
	aVel -= red
	bVel -= red

	aRps := aVel / p.mmPerRev
	bRps := bVel / p.mmPerRev

	p.Motors.SetRPS(aRps, bRps)
}

func (p *Platform) GetMaxVelocity() float32 {
	max := p.Motors.GetMaxRPS()
	return max * p.mmPerRev
}

func (p *Platform) GetMaxOmega() float32 {
	deltaV := p.GetMaxVelocity()
	return deltaV * 4 / p.wheelbase
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

func NewPlatform(/* Some config */) (*Platform, error) {
	_, err := host.Init()
	if err != nil {
		log.Fatal(err)
	}

	b, err := i2creg.Open("")
	if err != nil {
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
	}

	p.Motors = motor.NewMotors(dev)

	p.Camera = picamera.NewCamera(16, 16, 60)
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

	return p, nil
}

func (p *Platform) Update() error {
	pkts, err := p.dev.Poll()
	if err != nil {
		return err
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
