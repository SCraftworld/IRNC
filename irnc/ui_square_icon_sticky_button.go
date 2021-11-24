package irnc

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"sync"
	"time"
)

const MinTappedDuration = time.Second

// intented only for touchscreen, so no hover logic
type SquareIconStickyButton struct {
	widget.BaseWidget
	
	Icon fyne.Resource
	MinDim float32
	PaddingDim float32
	
	OnTapped func(*sync.WaitGroup) `json:"-"`
	tapVisual func()
	untapVisual func()
	tapMtx sync.Mutex
}

// Creates a new SquareIconStickyButton widget with icon and tap/untap handler
func NewSquareIconStickyButton(minDim, padding float32, icon fyne.Resource, tapped func(*sync.WaitGroup)) *SquareIconStickyButton {
	button := &SquareIconStickyButton{
		Icon: icon,
		MinDim: minDim,
		PaddingDim: padding,
		OnTapped: tapped,
	}
	
	button.ExtendBaseWidget(button)
	return button
}

// Link widget to its renderer
func (b *SquareIconStickyButton) CreateRenderer() fyne.WidgetRenderer {
	b.ExtendBaseWidget(b)
	icon := canvas.NewImageFromResource(b.Icon)
	icon.FillMode = canvas.ImageFillContain
	border := canvas.NewRectangle(color.White)
	background := canvas.NewRectangle(color.Black)
	b.tapVisual = func() {
		border.FillColor = &color.NRGBA{255, 0, 0, 255}
		canvas.Refresh(border)
	}
	b.untapVisual = func() {
		border.FillColor = color.White
		canvas.Refresh(border)
	}
	r := &buttonRenderer{
		icon: icon,
		background: background,
		border: border,
		button: b,
		layout: layout.NewHBoxLayout(),
	}
	return r
}

// Minimal size
func (b *SquareIconStickyButton) MinSize() fyne.Size {
	return fyne.Size{b.MinDim, b.MinDim}
}

// Tapped event handler
func (b *SquareIconStickyButton) Tapped(*fyne.PointEvent) {
	if b.OnTapped == nil { return }
	b.tapMtx.Lock()
	if b.tapVisual != nil {
		b.tapVisual()
	}
	b.Refresh()
	minTappedEndTime := time.Now().Add(MinTappedDuration)
	var wg sync.WaitGroup
	wg.Add(1)
	go func(){
		b.OnTapped(&wg)
		time.Sleep(time.Until(minTappedEndTime))
		wg.Wait()
		b.untapVisual()
		b.tapMtx.Unlock()
	}()
}

type buttonRenderer struct {
	icon *canvas.Image
	background *canvas.Rectangle
	border *canvas.Rectangle
	button *SquareIconStickyButton
	layout fyne.Layout
}

// Layout
func (r *buttonRenderer) Layout(size fyne.Size) {
	padding := r.button.PaddingDim
	border := float32(2)
	
	pos := float32(0)
	dim := size.Width
	if size.Height > dim {
		dim = size.Height
	}
	r.border.Move(fyne.NewPos(pos, pos))
	r.border.Resize(fyne.NewSize(dim, dim))
	
	pos += border
	dim -= 2 * border
	r.background.Move(fyne.NewPos(pos, pos))
	r.background.Resize(fyne.NewSize(dim, dim))
	
	pos += padding
	dim -= 2 * padding
	r.icon.Move(fyne.NewPos(pos, pos))
	r.icon.Resize(fyne.NewSize(dim, dim))
}

// Minimal size
func (r *buttonRenderer) MinSize() fyne.Size {
	return r.button.MinSize()
}

// Get canvas objects
func (r *buttonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.border, r.background, r.icon}
}

// Refresh button
func (r *buttonRenderer) Refresh() {
	r.icon.Refresh()
	r.background.Refresh()
	r.border.Refresh()
	r.Layout(r.button.Size())
}

// Destroy
func (r *buttonRenderer) Destroy() {
	return
}
