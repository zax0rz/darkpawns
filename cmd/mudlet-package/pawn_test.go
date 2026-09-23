package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The pawn in the dock's lockup picture is checked against the site
// header's canonical drawing: these read the five shapes from Header.astro
// and say whether a point is inside one.

// headerPath is the canonical pawn, relative to the package directory.
const headerPath = "../website-astro/src/components/Header.astro"

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
