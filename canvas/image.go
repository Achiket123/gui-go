package canvas

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/achiket123/gui-go/render"
)

// Image wraps a GPU texture and its pixel dimensions.
// Create with LoadImage, LoadImageFromBytes, or NewImageFromPixels.
type Image struct {
	id render.TextureID
	w  int
	h  int
	r  render.Renderer // back-reference for Dispose
}

// Width returns the image width in pixels.
func (img *Image) Width() int { return img.w }

// Height returns the image height in pixels.
func (img *Image) Height() int { return img.h }

// Dispose frees the GPU texture. Call when the image is no longer needed.
func (img *Image) Dispose() {
	if img.r != nil {
		img.r.DeleteTexture(img.id)
		img.r = nil
	}
}

// TextureID returns the opaque renderer texture handle.
func (img *Image) TextureID() render.TextureID { return img.id }

// LoadImage loads a PNG or JPEG from disk, uploads to GPU, and returns an Image.
func LoadImage(r render.Renderer, path string) (*Image, error) {
	f, err := os.Open(path) // #nosec G304 — intentional user-controlled path
	if err != nil {
		return nil, fmt.Errorf("canvas.LoadImage: %w", err)
	}
	defer f.Close()
	return decodeToImage(r, f)
}

// LoadImageFromBytes decodes PNG or JPEG from memory.
func LoadImageFromBytes(r render.Renderer, data []byte) (*Image, error) {
	return LoadImageFromReader(r, bytes.NewReader(data))
}

func decodeToImage(r render.Renderer, f interface{ Read([]byte) (int, error) }) (*Image, error) {
	type reader interface {
		Read([]byte) (int, error)
	}
	img, _, err := image.Decode(f.(reader))
	if err != nil {
		return nil, err
	}
	return imageFromGoImage(r, img)
}

// LoadImageFromReader decodes from any io.Reader.
func LoadImageFromReader(r render.Renderer, rd interface {
	Read([]byte) (int, error)
}) (*Image, error) {
	img, _, err := image.Decode(rd)
	if err != nil {
		return nil, err
	}
	return imageFromGoImage(r, img)
}

func imageFromGoImage(r render.Renderer, src image.Image) (*Image, error) {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBAModel.Convert(src.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.RGBA)
			i := (y*w + x) * 4
			pixels[i+0] = c.R
			pixels[i+1] = c.G
			pixels[i+2] = c.B
			pixels[i+3] = c.A
		}
	}
	id := r.UploadTexture(w, h, pixels)
	return &Image{id: id, w: w, h: h, r: r}, nil
}

// NewImageFromPixels creates an image from raw RGBA pixel data.
func NewImageFromPixels(r render.Renderer, w, h int, pixels []byte) *Image {
	id := r.UploadTexture(w, h, pixels)
	return &Image{id: id, w: w, h: h, r: r}
}
