// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package algo

import (
	"image"
	"math"
)

func findMinMaxRowwise(img *image.Gray) []image.Point {
	ret := make([]image.Point, img.Bounds().Dy())
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	cpp := 1

	for i := 0; i < h; i++ {
		var min, max uint8 = 255, 0
		row := img.Pix[img.Stride * i : img.Stride * i + w]
		for j := 0; j < w * cpp; j += cpp {
			if row[j] < min {
				min = row[j]
			}
			if row[j] > max {
				max = row[j]
			}
		}
		ret[i] = image.Pt(int(max), int(min))
	}

	return ret
}

// TODO: What fudge-factor do we need here?
const fudge = 80

/* FIXME: Out of sync with expandContrastAndThresh
func expandContrast(img *image.Gray, minMax []image.Point) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	cpp := 1

	for i := 0; i < h; i++ {
		scale := float32(0.0)
		if minMax[i].X - minMax[i].Y > fudge {
			scale = 255.0 / float32(minMax[i].X - minMax[i].Y)
		}

		row := img.Pix[img.Stride * i : img.Stride * i + w]
		for j := 0; j < w * cpp; j += cpp {
			newVal := float32(row[j] - uint8(minMax[i].Y)) * scale
			row[j] = uint8(newVal)
		}
	}
}
*/

func expandContrastAndThresh(img *image.Gray, minMax []image.Point) {
	w, _ := img.Bounds().Dx(), img.Bounds().Dy()
	cpp := 1

	for i, p := range minMax {
		diff := p.X - p.Y
		scale := float32(0.0)
		if p.X - p.Y > fudge {
			scale = 255.0 / float32(diff)
		}

		row := img.Pix[img.Stride * i : img.Stride * i + w]
		for j := 0; j < w * cpp; j += cpp {
			newVal := float32(row[j] - uint8(p.Y)) * scale
			if newVal > 127.0 {
				row[j] = 255
			} else {
				row[j] = 0
			}
		}
	}
}

func findMiddles(img *image.Gray) [][]float32 {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	middles := make([][]float32, h)

	e := float32(1.0) / float32(w)

	for y := 0; y < h; y++ {
		middles[y] = make([]float32, 0, 2)
		row := img.Pix[img.Stride * y : img.Stride * y + w]
		in := false
		start := 0

		for x, v := range row {
			if v == 0 {
				if in {
					if (x - start > 1) {
						middles[y] = append(middles[y], float32(start + x) * e / 2)
					}
					in = false
				}
			} else {
				if !in {
					start = x
					in = true
				}
			}
		}
		if in {
			if (w - start >= 2) {
				middles[y] = append(middles[y], float32(start + w) * e / 2)
			}
		}
	}
	return middles
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

type Point struct {
	U, V float32
}

func findClosest(points []float32, pt float32) int {
	min := 4.0
	mindx := 0
	for i, p := range points {
		dst := math.Abs(float64(pt - p))
		if dst < min {
			min = dst
			mindx = i
		}
	}
	return mindx
}

func FindLine(img *image.Gray) []float32 {
	minMax := findMinMaxRowwise(img)
	expandContrastAndThresh(img, minMax)
	middles := findMiddles(img)

	dx := float32(0.0)
	linePoints := make([]float32, len(middles))

	var current float32
	for _, row := range middles {
		if len(row) == 0 {
			continue
		}
		current = row[0]
	}

	for i, row := range middles {
		if len(row) == 0 {
			linePoints[i] = float32(math.NaN())
			continue
		}
		pred := current + dx
		idx := findClosest(row, pred)
		dx = row[idx] - current
		linePoints[i] = row[idx] - 0.5
	}

	return linePoints
}
