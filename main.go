package main

import (
	"fmt"
	"math"
	"math/rand"
	"path"
	"sync"
	"time"

	"github.com/tfriedel6/canvas"
	"github.com/tfriedel6/canvas/sdlcanvas"
)

// --------- Configuration ---------
const (
	width        = 1024
	height       = 768
	particleSize = 2.5
	friction     = 0.4159 // 0.5 = High viscosity (microscope liquid)
)

type particle struct {
	x, y   float64
	vx, vy float64
	role   string // "green", "red", "yellow"
}

var (
	particles  []*particle
	cv         *canvas.Canvas
	wg         sync.WaitGroup
	frameCount int
)

// --------- Helpers ---------
func hsvToHex(h, s, v float64) string {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 220:
		r, g, b = 0, x, c
	case h < 360:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return fmt.Sprintf("#%02X%02X%02X", int((r+m)*255), int((g+m)*255), int((b+m)*255))
}

func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// --------- Particle Logic ---------

func createGroup(count int, role string) []*particle {
	group := make([]*particle, count)
	for i := 0; i < count; i++ {
		p := &particle{
			x:    rand.Float64() * width,
			y:    rand.Float64() * height,
			role: role,
		}
		group[i] = p
		particles = append(particles, p)
	}
	return group
}

func applyRule(group1 []*particle, group2 []*particle, g float64) {
	wg.Add(len(group1))
	for i := 0; i < len(group1); i++ {
		go func(idx int) {
			defer wg.Done()
			p1 := group1[idx]
			var fx, fy float64
			for _, p2 := range group2 {
				dx := p1.x - p2.x
				dy := p1.y - p2.y
				d := math.Sqrt(dx*dx + dy*dy)

				if d > 0 && d < 80 {
					// Basic Force calculation
					f := g * 1.0 / d
					fx += f * dx
					fy += f * dy
				}
			}
			p1.vx = (p1.vx + fx) * friction
			p1.vy = (p1.vy + fy) * friction
		}(i)
	}
	wg.Wait()
}

func updatePhysics(p *particle) {
	p.x += p.vx
	p.y += p.vy

	// Boundary Check
	if p.x <= 0 || p.x >= width {
		p.vx *= -1
	}
	if p.y <= 0 || p.y >= height {
		p.vy *= -1
	}
}

func getMicroscopeColor(p *particle) string {
	speed := math.Sqrt(p.vx*p.vx + p.vy*p.vy)
	glow := math.Min(speed*10, 1.0)

	switch p.role {
	case "green":
		return hsvToHex(130, 0.9, 0.4+(glow*0.6)) // GFP (Green)
	case "red":
		return hsvToHex(350, 0.9, 0.5+(glow*0.5)) // mCherry (Red)
	case "yellow":
		return hsvToHex(50, 0.8, 0.3+(glow*0.7)) // YFP (Yellow)
	}
	return "#FFFFFF"
}

// --------- Main Loop ---------

func main() {
	wnd, canvasObj, err := sdlcanvas.CreateWindow(width, height, "Bio-Digital Microscopy")
	if err != nil {
		panic(err)
	}
	cv = canvasObj

	fontPath := path.Join("assets", "fonts", "montserrat.ttf")
	cv.SetFont(fontPath, 14)

	// Create Biological Agents
	green := createGroup(3800, "green")   // The "Cells"
	red := createGroup(5300, "red")       // The "Predators/Enzymes"
	yellow := createGroup(3600, "yellow") // The "Neural/Connective"

	wnd.MainLoop(func() {
		frameCount++
		start := time.Now()

		// Background: Deep dark navy instead of pure black for depth
		cv.SetFillStyle("#00050A")
		cv.FillRect(0, 0, width, height)

		// rules (still based on groups)
		// apply Fibonacci values, scaled down to stay stable

		fib5 := float64(Fibonacci(6)) // 5
		fib3 := float64(Fibonacci(9)) // 2
		fib1 := float64(Fibonacci(9)) // 1

		scale := 0.05 // reduces intensity so the forces remain in [-1,1] range

		applyRule(red, red, fib5*scale*-1)  // slight repulsion, prevents solid red cores
		applyRule(red, green, fib3*scale)   // mild attraction for interplay with green
		applyRule(red, yellow, -fib3*scale) // balanced repulsion/attraction for vortex flow

		applyRule(yellow, yellow, fib1*scale) // mild self-attraction forms yellow filaments
		applyRule(yellow, green, -0.20)       // gentle repulsion, stabilizes green-yellow interface
		applyRule(yellow, red, -0.15)         // light repulsion, avoids yellow-red clumping
		applyRule(green, green, -fib5*scale)  // strong self-repulsion creates green voids
		applyRule(green, red, -fib3*scale)
		applyRule(green, yellow, 0.20) // gentle attraction, stabilizes green-yellow interface

		// 2. Render
		for _, p := range particles {
			updatePhysics(p)
			cv.SetFillStyle(getMicroscopeColor(p))
			// Use circular-ish rects for organic look
			cv.FillRect(p.x, p.y, particleSize, particleSize)
		}

		// 3. UI
		cv.SetFillStyle("#FFFFFF")
		cv.FillText(fmt.Sprintf("Agents: %d | Time: %d", len(particles), frameCount), 10, 20)

		dt := time.Since(start)
		if dt > 0 {
			cv.FillText(fmt.Sprintf("FPS: %d", int(1.0/dt.Seconds())), 10, 40)
		}
	})
}
