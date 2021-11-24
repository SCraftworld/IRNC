package irnc

import (
	"fyne.io/fyne/v2"
)

// Minimal container in middle and evenly maxed containers on sides
type irncLayout struct { }

// Minimal size
func (l *irncLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var w, h float32
	for _, o := range objects {
		chMinSize := o.MinSize()
		w += chMinSize.Width
		if chMinSize.Height > h {
			h = chMinSize.Height
		}
	}
	return fyne.NewSize(w, h)
}

// Layout function
func (l *irncLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) != 3 { return }
	left := objects[0]
	mid := objects[1]
	right := objects[2]
	
	midSize := mid.MinSize()
	midLeft := (containerSize.Width - midSize.Width)/2
	
	mid.Resize(fyne.NewSize(midSize.Width, containerSize.Height))
	mid.Move(fyne.NewPos(midLeft, 0))
	
	left.Resize(fyne.NewSize(midLeft, containerSize.Height))
	left.Move(fyne.NewPos(0, 0))
	
	right.Resize(fyne.NewSize(containerSize.Width - midLeft - midSize.Width, containerSize.Height))
	right.Move(fyne.NewPos(midLeft+midSize.Width, 0))
}
