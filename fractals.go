package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"os"
	"math/rand"
	"strconv"
	"log"
)

type FractalConfig struct {
	Zoom           float64
	CenterPoint    Point
	MaxValue       float64
	Start          complex128
	MaxIterations  int
	DisplayQuality image.Rectangle
	Seed           int64
	Process        int
	FileName       string
}

type progress int

type chaoticIterator func(accum, start complex128) complex128

type Point struct {
	X, Y float64
}

type Rectangle struct {
	Min, Max Point
}

func (r Rectangle) Dx() float64 {
	return math.Abs(r.Min.X - r.Max.X)
}

func (r Rectangle) Dy() float64 {
	return math.Abs(r.Min.Y - r.Max.Y)
}

func CalcColor(val float64, seed *int64) color.RGBA {

	/* Predictably Randomize */
	rand.Seed(*seed)                                                                                                // Predictably scramble the seed
	seedScrambled := rand.Int63() % 1000000                                                                        // Returns an int64, but only 6 digits

	// val range == 0.0 - 1.0
	if (seedScrambled%2)==0 { // Flip a coin for colored/black theme
		if val == 1.0 {
			// Black
			return color.RGBA{0, 0, 0, uint8((255 + (seedScrambled*100/999999)+512)/4)} 				// Alpha is the seed scaled to 255
		}
	}

	// Extract color from seed
	seedR := (seedScrambled % 100) % 100
	seedG := (seedScrambled / 100) % 100
	seedB := (seedScrambled / 10000) % 100

	/* This is what the above looks like using slices... above still easier but which performs? */
	//seedSlice := make([]byte, 6)											// Prepare a slice of bytes to hold our scrambled value
	//binary.BigEndian.PutUint64(seedSlice, uint64(seedScrambled))							// Puts our scrambled value into a uint64
	//seedR := binary.BigEndian.Uint16(seedSlice[0:1])	// 123456 => 56, then uint to int
	//seedG := binary.BigEndian.Uint16(seedSlice[2:3])	// 123456 => 34
	//seedB := binary.BigEndian.Uint16(seedSlice[4:5])	// 123456 => 12

	// Normalize (from 0-99 to 0-255)
	max := int64(255)
	div := int64(100)
	seedR = seedR * max / div
	seedG = seedG * max / div
	seedB = seedB * max / div

	val = val * val * val                                                                                                // "enhance"

	// Color to return
	a := uint8((seedR + seedB + seedG) / 3);
	r := uint8(math.Pow(float64(seedR), val))
	g := uint8(math.Pow(float64(seedG), val))
	b := uint8(math.Pow(float64(seedB), val))

	color := color.RGBA{r, g, b, a}
	return color
}

func GetEscapeIterations(x, y float64, maxIteration int, givenFunc chaoticIterator, error float64, start complex128) int {
	iterated := complex(x, y)
	iteration := 1

	for cmplx.Abs(iterated) < error && iteration < maxIteration {
		iterated = givenFunc(iterated, start)
		iteration++
	}
	return iteration
}

func Man(val, val0 complex128) complex128 {
	return val * val + val0
}

func fractal1(val, val0 complex128) complex128 {
	return val * cmplx.Sqrt(cmplx.Cosh(val * val * val)) * val0
}

func fractal2(val, val0 complex128) complex128 {
	return cmplx.Sqrt(cmplx.Sinh(val)) + val0
}

func fractal3(val, val0 complex128) complex128 {
	return val * val * cmplx.Exp(val) + val0
}

func fractal4(val, val0 complex128) complex128 {
	return val * val * val * val * val + val0
}

func fractal5(val, val0 complex128) complex128 {
	return (val * val + val) / cmplx.Log(val) + val0
}
func CreateFractalImage(fc FractalConfig, canceler <-chan bool) <-chan int {
	// Return a read-only channel for pct, read from a read-only channel canceler
	// canceler is a chan passed in by main, boolean will be true when cencel requested
	// out is the channel for returning the percentage, int
	out := make(chan int) // Buffered so we can send pct while it works
	quit := false
	go func() {
		// This is a go func so it can return the channel before blocking
		viewing_window := Rectangle{
			Point{-0.540 / fc.Zoom + fc.CenterPoint.X, -0.960 / fc.Zoom + fc.CenterPoint.Y}, // ToDo - use values from fc struct instead of hardcoded values
			Point{0.540 / fc.Zoom + fc.CenterPoint.X, 0.960 / fc.Zoom + fc.CenterPoint.Y},
		}
		lastPct := 1
		currentPct := 1
		out <- 1

		/* Set which fractal algorithm */
		algorithm := fractal1
		if fc.Process == 1 {
			algorithm = fractal1
		}
		if fc.Process == 2 {
			algorithm = Man
		}
		if fc.Process == 3 {
			algorithm = fractal4
		}
		if fc.Process == 4 {
			algorithm = fractal3
		}

		// Calculate Color Value
		log.Print("GOPHER " + strconv.Itoa(fc.Process) + " STAGE 1")                                                // STAGE 1 - CALCULATE COLORS
		im := image.NewRGBA(fc.DisplayQuality)
		hist := make([]int, fc.MaxIterations)                                                                        // slice of int with len and cap = 100

		dx := viewing_window.Dx() / float64(fc.DisplayQuality.Dx())
		dy := viewing_window.Dy() / float64(fc.DisplayQuality.Dy())

		/* Loop over each pixel and adjust to viewing window */
		for x := 0; x < fc.DisplayQuality.Dx(); x++ {
			for y := 0; y < fc.DisplayQuality.Dy(); y++ {
				adjustedX := float64(x) * dx + viewing_window.Min.X
				adjustedY := float64(y) * dy + viewing_window.Min.Y
				hist[GetEscapeIterations(adjustedX, adjustedY, fc.MaxIterations, algorithm, fc.MaxValue, fc.Start) - 1] += 1
			}

			select {
			case <-canceler:
			// Catch cancel
				log.Print("Received canceler STG1!")
				quit = true
				return
			default:
				continue
			}

			currentPct = (x * 100) / (fc.DisplayQuality.Dx() + 1)                                                // Send pct completed
			if (lastPct < currentPct) {
				out <- currentPct
				lastPct = currentPct
			}
		}

		log.Print("\tGOPHER " + strconv.Itoa(fc.Process) + " STAGE 2")                                                // STAGE 2 - CALC HISTOGRAM
		// Setup and calculate the histogram
		// --> vals = percentage of pixels below the current val in terms of iterations
		vals := make([]float64, fc.MaxIterations)
		total := 0
		last_hist_val := 0.0
		lastPct = 0

		for i, h := range hist {
			// for key, val := range item
			total += h
			last_hist_val = float64(h)
			select {
			case <-canceler:
			// Catch cancel
				log.Print("Received canceler!")
				quit = true
				return
			default:
				continue
			}
			out <- i                                                                                        // Update PCT - no need to limit output
		}

		log.Print("\t\tGOPHER " + strconv.Itoa(fc.Process) + " STAGE 3")                                                // STAGE 3 - HISTOGRAM PASS#2
		lastPct = 0

		vals[0] = float64(hist[0]) / float64(total)
		for v := 1; v < len(vals) - 1; v++ {
			vals[v] = vals[v - 1] + float64(hist[v]) / (float64(total) - last_hist_val)
			currentPct = (v * (len(vals) - 1)) / 100
			if (lastPct < currentPct) {
				// Update PCT
				out <- currentPct
				lastPct = currentPct
			}

			// Catch cancel, break and set the quit flag
			select {
			case <-canceler:
			// Catch cancel
				log.Print("Received canceler!")
				quit = true
				return
			default:
				continue
			}
		}

		vals[len(vals) - 1] = 1.0

		log.Print("\t\t\tGOPHER " + strconv.Itoa(fc.Process) + " STAGE 4")                                        // STAGE 4 - PIXELS
		lastPct = 0
		// Get the actual pixel values and assign them to the image
		for x := 0; x < fc.DisplayQuality.Dx(); x++ {
			for y := 0; y < fc.DisplayQuality.Dy(); y++ {
				adjustedX := float64(x) * dx + viewing_window.Min.X
				adjustedY := float64(y) * dy + viewing_window.Min.Y
				val := vals[GetEscapeIterations(adjustedX, adjustedY, fc.MaxIterations, algorithm, fc.MaxValue, fc.Start) - 1]
				col := CalcColor(val, &fc.Seed)
				im.SetRGBA(x, fc.DisplayQuality.Dy() - y - 1, col)
			}
			// Output new percentage, if it has changed
			currentPct = (x * 100 / fc.DisplayQuality.Dx())
			if (lastPct < currentPct) {
				// Update PCT
				out <- currentPct
				lastPct = currentPct
			}
			// Catch cancel, break and set the quit flag
			select {
			case <-canceler:
			// Catch cancel
				log.Print("Received canceler!")
				quit = true
				return
			default:
				continue
			}
		}
		if !quit {
			imageFile, err := os.Create("img/" + fc.FileName)
			if err == nil {
				defer imageFile.Close()
				png.Encode(imageFile, im)
			} else {
				fmt.Println("failed")
			}

			// Total PCT should be == displayQuality.Dx * Dy + max_iterations for 3 stages combined
			if (currentPct < 100) {
				out <- 100
			}
		}
		return                                                                                                        // Kill the gopher
	}()
	return out // Return channel to the caller
}


/* Generate Fractal Image */
func generateFractal(p int, seed int64, canceler <-chan bool, pct chan <- int) {
	// Receive only canceler, send only pct
	go func() {
		/* Initialize 4 Fractal configurations */
		fc := &FractalConfig{
			MaxIterations: 100, // ToDo - Currently, routine depends on this being 100
			MaxValue: 3.0,
			DisplayQuality : image.Rect(10, 10, 540, 960),
			Process : p,
			Start : complex(float64(0.8), float64(0.6)), //complex(float64(0.8 + float64(p / 10)), float64(0.6 + float64(p / 10))),
			Zoom : 7.25, // Image Zoom Level
			CenterPoint : Point{-0.8, 0.425}, // Image Center Point
			Seed : seed, // 6 digit random seed
			FileName : strconv.Itoa(p) + "_" + strconv.FormatInt(seed, 10) + ".png", // Output file relative path+name
		}
		if (p == 2) {
			fc.Start = complex(float64(0.35), float64(-0.1))
			fc.MaxValue = 1.5
			fc.Zoom = 5.5
			fc.CenterPoint = Point{0.4, -1.105}
		}
		if (p == 3) {
			fc.Zoom = 740.0
			fc.CenterPoint = Point{-0.5, -0.095}
		}
		if (p == 4) {
			fc.Start = complex(float64(0.15), float64(0.15))
			fc.MaxValue = 2.5
			fc.Zoom = 6.5
			fc.CenterPoint = Point{-0.2, -1.05}
		}

		/* GO! */
		c := CreateFractalImage(*fc, canceler)                                                                        // start go routine, pass in our canceler channel

		seconds := float64(fc.MaxIterations * fc.DisplayQuality.Dx() * fc.DisplayQuality.Dy() * 28) / (10 * 540 * 960 * 4)        // Estimate creation time
		fmt.Printf("Expected creation time: %.3f seconds (%.5f minutes)\n", seconds, seconds / 60)
		done := false
		for {
			select {
			case cpct := <-c:
			// get current percent from the generateFractalImage function
				if (cpct == 100 && done == false) {
					pct <- 100
					log.Print("Done with " + strconv.Itoa(p))
					done = true
					cpct = 0
				} else {
					pct <- cpct                                                                        // Pass the message along to the pct channel
				}
				break
			case <-canceler:
				log.Print("Received canceler!")
				done = true
				break
			default:
				continue
			}
			if done {
				break
			}
		}
		wg.Done()                                                                                                // Decrement waitgroup
	}()
}