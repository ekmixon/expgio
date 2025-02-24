// SPDX-License-Identifier: Unlicense OR MIT

package main

import (
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"

	"github.com/egonelbre/expgio/f32color"
	"github.com/fogleman/ln/ln"
)

func main() {
	mesh, err := ln.LoadOBJ("suzanne.obj")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	mesh.UnitCube()

	th := material.NewTheme(gofont.Collection())
	ui := &UI{
		Theme: th,
		Mesh:  mesh,
	}
	go func() {
		w := app.NewWindow(app.Title("Slicer"))
		if err := ui.Run(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	app.Main()
}

type UI struct {
	Theme *material.Theme

	Mesh *ln.Mesh
}

func (ui *UI) Run(w *app.Window) error {
	var ops op.Ops

	for e := range w.Events() {
		switch e := e.(type) {
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			ui.Layout(gtx)
			e.Frame(gtx.Ops)

		case key.Event:
			switch e.Name {
			case key.NameEscape:
				return nil
			}

		case system.DestroyEvent:
			return e.Err
		}
	}

	return nil
}

func (ui *UI) Layout(gtx layout.Context) layout.Dimensions {
	return Slice{
		Mesh: ui.Mesh,
		RotY: float64(gtx.Now.UnixNano()/1e6) * 0.001,
		Eye:  ln.Vector{-0.5, 0.5, 2},
		Up:   ln.Vector{0, 1, 0},

		Slices: 128,
	}.Layout(gtx)
}

type Scene struct {
	Mesh   *ln.Mesh
	RotY   float64
	Eye    ln.Vector
	Up     ln.Vector
	Center ln.Vector
}

func (scene Scene) Layout(gtx layout.Context) layout.Dimensions {
	op.InvalidateOp{}.Add(gtx.Ops)

	size := gtx.Constraints.Max

	s := ln.Scene{}
	if scene.RotY != 0 {
		s.Add(ln.NewTransformedShape(scene.Mesh, ln.Rotate(ln.Vector{0, 1, 0}, scene.RotY)))
	} else {
		s.Add(scene.Mesh)
	}

	paths := s.Render(scene.Eye, scene.Center, scene.Up, float64(size.X), float64(size.Y), 35, 0.1, 100, 0.01)

	defer op.Save(gtx.Ops).Load()

	p := clip.Path{}
	p.Begin(gtx.Ops)

	h := float32(size.Y)
	for _, path := range paths {
		p.MoveTo(f64pt(path[0], h))
		for _, v := range path[1:] {
			p.LineTo(f64pt(v, h))
		}
	}

	clip.Stroke{
		Path:  p.End(),
		Width: 1,
	}.Op().Add(gtx.Ops)

	paint.ColorOp{Color: color.NRGBA{A: 0xFF}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{
		Size: size,
	}
}

type Slice struct {
	Mesh *ln.Mesh

	Slices int

	RotY   float64
	Eye    ln.Vector
	Up     ln.Vector
	Center ln.Vector
}

func (scene Slice) Layout(gtx layout.Context) layout.Dimensions {
	op.InvalidateOp{}.Add(gtx.Ops)

	size := gtx.Constraints.Max

	aspect := float64(size.X) / float64(size.Y)
	rotation := ln.Rotate(ln.Vector{0, 1, 0}, scene.RotY).Scale(ln.Vector{0.5, 0.5, 0.5})
	matrix := ln.LookAt(scene.Eye, scene.Center, scene.Up)
	matrix = matrix.Perspective(35, aspect, 0.1, 100)

	for i := 0; i < scene.Slices; i++ {
		func() {
			slice := float64(i)/float64(scene.Slices)*2 - 1

			point := ln.Vector{0, slice, 0}
			plane := ln.Plane{point, ln.Vector{0, 1, 0}}
			paths := plane.IntersectMesh(scene.Mesh)
			paths = paths.Simplify(1e-6)

			// rendering
			paths = paths.Transform(rotation)
			paths = paths.Transform(matrix)
			paths = paths.Transform(
				ln.Translate(ln.Vector{1, 1, 0}).
					Scale(ln.Vector{X: float64(size.X) / 2, Y: float64(size.Y) / 2, Z: 0}),
			)

			defer op.Save(gtx.Ops).Load()

			p := clip.Path{}
			p.Begin(gtx.Ops)

			h := float32(size.Y)
			for _, path := range paths {
				p.MoveTo(f64pt(path[0], h))
				for _, v := range path[1:] {
					p.LineTo(f64pt(v, h))
				}
			}

			clip.Stroke{
				Path:  p.End(),
				Width: 3,
			}.Op().Add(gtx.Ops)

			paint.ColorOp{
				Color: f32color.HSL(float32(slice), 0.6, 0.6),
			}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}()
	}

	return layout.Dimensions{
		Size: size,
	}
}

func f64pt(v ln.Vector, h float32) f32.Point {
	return f32.Point{
		X: float32(v.X),
		Y: h - float32(v.Y),
	}
}
