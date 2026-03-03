package main

import (
	"fmt"
	"math"
	"math/rand"
	"path"
	"sync"

	"github.com/tfriedel6/canvas/sdlcanvas"
)

const (
	width        = 1024
	height       = 768
	particleSize = 2.0
	friction     = 0.45
	gridSize     = 40 // Cells for spatial partitioning
	numWorkers   = 8  // Parallel processing threads
)

// --------- Types & Globals ---------

type particle struct {
	x, y, vx, vy float64
	role         int // 0: green, 1: red, 2: yellow
	glow         float64
}

// Grid partitions the screen to avoid O(N^2) checks
type Grid [width/gridSize + 1][height/gridSize + 1][]*particle

var (
	particles  []*particle
	spatial    Grid
	juliaCRe   = -0.7
	juliaCIm   = 0.27015
	colors     = []string{"#32CD32", "#FF4500", "#FFD700"}
	frameCount int
)

// --------- Spatial Grid Logic ---------

func (g *Grid) clear() {
	for i := range g {
		for j := range g[i] {
			g[i][j] = g[i][j][:0]
		}
	}
}

func (g *Grid) add(p *particle) {
	gx, gy := int(p.x/gridSize), int(p.y/gridSize)
	if gx >= 0 && gx < len(g) && gy >= 0 && gy < len(g[0]) {
		g[gx][gy] = append(g[gx][gy], p)
	}
}

// --------- Physics Engine ---------

func getInteraction(r1, r2 int) float64 {
	// Interaction Matrix: [Self][Target]
	rules := [3][3]float64{
		{-0.25, -0.15, 0.10}, // Green
		{0.15, -0.35, -0.10}, // Red
		{-0.05, -0.05, 0.25}, // Yellow
	}
	return rules[r1][r2]
}

func updateWorker(start, end int, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := start; i < end; i++ {
		p1 := particles[i]
		gx, gy := int(p1.x/gridSize), int(p1.y/gridSize)

		var fx, fy float64
		neighbors := 0

		// Check 3x3 grid neighborhood
		for nx := gx - 1; nx <= gx+1; nx++ {
			for ny := gy - 1; ny <= gy+1; ny++ {
				if nx < 0 || nx >= len(spatial) || ny < 0 || ny >= len(spatial[0]) {
					continue
				}
				for _, p2 := range spatial[nx][ny] {
					dx := p2.x - p1.x
					dy := p2.y - p1.y
					distSq := dx*dx + dy*dy

					if distSq > 0 && distSq < 2500 { // 50px radius
						neighbors++
						d := math.Sqrt(distSq)
						g := getInteraction(p1.role, p2.role)
						fx += (g * dx) / d
						fy += (g * dy) / d
					}
				}
			}
		}

		// Fractal Force
		zx := (p1.x - width/2) * 0.003
		zy := (p1.y - height/2) * 0.003
		iter := 0
		for i := 0; i < 12; i++ {
			xt := zx*zx - zy*zy + juliaCRe
			zy = 2*zx*zy + juliaCIm
			zx = xt
			if zx*zx+zy*zy > 4 {
				break
			}
			iter++
		}
		p1.glow = float64(iter) / 12.0

		// Apply physics
		p1.vx = (p1.vx + fx) * friction
		p1.vy = (p1.vy + fy) * friction

		// Julia suction if density is high
		if neighbors > 12 {
			p1.vx -= zx * 0.008
			p1.vy -= zy * 0.008
		}

		p1.x += p1.vx
		p1.y += p1.vy

		// Bounce
		if p1.x < 0 || p1.x > width {
			p1.vx *= -1
		}
		if p1.y < 0 || p1.y > height {
			p1.vy *= -1
		}
	}
}

// --------- Main Loop ---------

func main() {
	wnd, cv, err := sdlcanvas.CreateWindow(width, height, "Bio-Digital Grid v2")
	if err != nil {
		panic(err)
	}

	font := path.Join("assets", "fonts", "montserrat.ttf")
	cv.SetFont(font, 32)

	// Particle Population
	counts := []int{5000, 5000, 5000} // Total 15k
	for role, count := range counts {
		for i := 0; i < count; i++ {
			particles = append(particles, &particle{
				x:    rand.Float64() * width,
				y:    rand.Float64() * height,
				role: role,
			})
		}
	}

	wnd.MainLoop(func() {
		frameCount++

		// 1. Rebuild Grid
		spatial.clear()
		for _, p := range particles {
			spatial.add(p)
		}

		// 2. Parallel Physics Update
		var wg sync.WaitGroup
		chunkSize := len(particles) / numWorkers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			start := i * chunkSize
			end := (i + 1) * chunkSize
			if i == numWorkers-1 {
				end = len(particles)
			}
			go updateWorker(start, end, &wg)
		}
		wg.Wait()

		// 3. Rendering
		cv.SetFillStyle("#00050A")
		cv.FillRect(0, 0, width, height)

		for _, p := range particles {
			cv.SetFillStyle(colors[p.role])
			size := particleSize + (p.glow * 3)
			cv.FillRect(p.x, p.y, size, size)
		}

		// 4. UI & Fractal Evolution
		juliaCRe += 0.0001
		cv.SetFillStyle("#FFF")
		cv.FillText(fmt.Sprintf("FPS Check | Particles: %d | Frame: %d", len(particles), frameCount), 10, 25)
	})
}
