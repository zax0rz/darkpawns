package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The pawn is not redrawn for Mudlet. Mudlet releases up to 5.0 cannot put
// an SVG in a label, so the generator reads the five canonical shapes from
// the site header (the drawing check_wordmark.py enforces) and rasterizes
// them here. A change to the mark reaches the package the next time it is
// built, and TestPackageIsCurrent fails until it is.

// headerPath is the canonical pawn, relative to the package directory.
const headerPath = "../website-astro/src/components/Header.astro"

// pawnInk is DESIGN.md's Ink: the header draws the pawn in currentColor, and
// the header's currentColor is Ink.
var pawnInk = color.NRGBA{R: 0x1A, G: 0x16, B: 0x14, A: 0xFF}

// pawnHeight is the rendered height in pixels: twice the dock's 56-pixel
// header, so the mark stays crisp on high-density displays.
const pawnHeight = 112

type pawnShape struct {
	kind   string    // circle, rect, polygon
	values []float64 // cx cy r | x y width height | x1 y1 x2 y2 ...
}

type pawnDrawing struct {
	viewBox [4]float64
	shapes  []pawnShape
}

var (
	pawnSVG      = regexp.MustCompile(`(?s)<svg class="pawn" viewBox="([^"]+)"[^>]*>(.*?)</svg>`)
	pawnShapeTag = regexp.MustCompile(`<(circle|rect|polygon)\b([^>]*)>`)
	pawnAttr     = regexp.MustCompile(`([a-zA-Z-]+)\s*=\s*"([^"]*)"`)
)

// readPawn extracts the header's pawn drawing.
func readPawn(root string) (pawnDrawing, error) {
	raw, err := os.ReadFile(filepath.Clean(filepath.Join(root, headerPath)))
	if err != nil {
		return pawnDrawing{}, err
	}
	match := pawnSVG.FindSubmatch(raw)
	if match == nil {
		return pawnDrawing{}, fmt.Errorf("%s: no <svg class=\"pawn\">", headerPath)
	}
	var drawing pawnDrawing
	box := strings.Fields(string(match[1]))
	if len(box) != 4 {
		return pawnDrawing{}, fmt.Errorf("pawn viewBox %q", match[1])
	}
	for i, field := range box {
		if drawing.viewBox[i], err = strconv.ParseFloat(field, 64); err != nil {
			return pawnDrawing{}, err
		}
	}
	for _, tag := range pawnShapeTag.FindAllSubmatch(match[2], -1) {
		attrs := map[string]string{}
		for _, a := range pawnAttr.FindAllSubmatch(tag[2], -1) {
			attrs[string(a[1])] = string(a[2])
		}
		kind := string(tag[1])
		var fields []string
		switch kind {
		case "circle":
			fields = []string{attrs["cx"], attrs["cy"], attrs["r"]}
		case "rect":
			fields = []string{attrs["x"], attrs["y"], attrs["width"], attrs["height"]}
		case "polygon":
			fields = strings.FieldsFunc(attrs["points"], func(r rune) bool { return r == ',' || r == ' ' })
		}
		shape := pawnShape{kind: kind}
		for _, field := range fields {
			v, err := strconv.ParseFloat(field, 64)
			if err != nil {
				return pawnDrawing{}, fmt.Errorf("pawn %s: %w", kind, err)
			}
			shape.values = append(shape.values, v)
		}
		drawing.shapes = append(drawing.shapes, shape)
	}
	return drawing, nil
}

func (s pawnShape) contains(x, y float64) bool {
	v := s.values
	switch s.kind {
	case "circle":
		dx, dy := x-v[0], y-v[1]
		return dx*dx+dy*dy <= v[2]*v[2]
	case "rect":
		return x >= v[0] && x <= v[0]+v[2] && y >= v[1] && y <= v[1]+v[3]
	case "polygon":
		inside := false
		n := len(v) / 2
		for i, j := 0, n-1; i < n; j, i = i, i+1 {
			xi, yi, xj, yj := v[2*i], v[2*i+1], v[2*j], v[2*j+1]
			if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				inside = !inside
			}
		}
		return inside
	}
	return false
}

// renderPawnPNG draws the pawn in Ink on transparency, with 4x4 supersampled
// edges. Go's PNG encoder is deterministic, so the committed package only
// changes when the drawing does.
func renderPawnPNG(drawing pawnDrawing, height int) ([]byte, error) {
	vx, vy, vw, vh := drawing.viewBox[0], drawing.viewBox[1], drawing.viewBox[2], drawing.viewBox[3]
	scale := float64(height) / vh
	width := int(math.Round(vw * scale))
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	const samples = 4
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			covered := 0
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					x := vx + (float64(px)+(float64(sx)+0.5)/samples)/scale
					y := vy + (float64(py)+(float64(sy)+0.5)/samples)/scale
					for _, shape := range drawing.shapes {
						if shape.contains(x, y) {
							covered++
							break
						}
					}
				}
			}
			if covered > 0 {
				c := pawnInk
				c.A = uint8(covered * 255 / (samples * samples))
				img.SetNRGBA(px, py, c)
			}
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
