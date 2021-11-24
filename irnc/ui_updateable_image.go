package irnc

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"image"
)

// Widget with image which has capabilities for being updated/replaced
type UpdateableImage struct {
	widget.BaseWidget
	minSize fyne.Size
	img *canvas.Image
}

// Renderer for widget with updateable image
type updateableImageRenderer struct {
	updImg *UpdateableImage
}

// Minimal size
func (r *updateableImageRenderer) MinSize() fyne.Size {
	return r.updImg.MinSize()
}

// Objects slice
func (r *updateableImageRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.updImg.img}
}

// Refresh image
func (r *updateableImageRenderer) Refresh() {
	r.updImg.img.Refresh()
}

// Layout function
func (r *updateableImageRenderer) Layout(size fyne.Size) {
	r.updImg.img.Resize(size)
}

// Destruction function
func (r *updateableImageRenderer) Destroy() {
	r.updImg.img = nil
}

// Factory function for UpdateableImage widgets
func NewUpdateableImage(minSz fyne.Size) (*UpdateableImage) {
	ret := &UpdateableImage{}
	ret.ExtendBaseWidget(ret)
	ret.img = &canvas.Image{}
	ret.minSize = minSz
	ret.img.FillMode = canvas.ImageFillContain
	return ret
}

// Get renderer of UpdateableImage widget
func (i *UpdateableImage) CreateRenderer() fyne.WidgetRenderer {
	return &updateableImageRenderer{updImg: i}
}

// Update image in UpdateableImage widget
func (i *UpdateableImage) Update(img image.Image) {
	i.img.Image = img
	i.img.Refresh()
}

// Minimal size
func (i *UpdateableImage) MinSize() fyne.Size {
	return i.minSize
}
