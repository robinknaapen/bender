// Package appicon holds bender's application icon, shared by the
// platform backends (Windows AUMID registration, Linux tray pixmaps and
// notification fallback). Swap appicon.png to change the art.
package appicon

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
)

//go:embed appicon.png
var PNG []byte

// Decode returns the icon as an image.
func Decode() (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(PNG))
	return img, err
}
