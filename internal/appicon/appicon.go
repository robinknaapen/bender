// Package appicon holds bender's application icon, passed to spectacle
// (AUMID registration, tray pixmaps, notification fallback) at startup.
// Swap appicon.png to change the art.
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
