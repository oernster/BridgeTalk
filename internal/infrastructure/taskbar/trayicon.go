package taskbar

// The picture the Linux tray hands the desktop (FR-814).
//
// fyne.io/systray turns the PNG it is given into the ARGB bytes it publishes by keeping the low
// byte of each 16 bit premultiplied channel. That is right only where a pixel is fully opaque or
// fully clear; anywhere between, the byte it keeps is noise. The artwork is almost nowhere fully
// opaque (most of its body sits a few steps under), so handed the icon as it stands the tray drew
// the outline filled with coloured static. Each pixel is therefore made fully opaque or fully clear
// before the library sees it, which its conversion carries exactly and which a corrected conversion
// would carry exactly too. The colours are the artwork's own; only the partial coverage at the edge
// goes; the desktop scaling the picture down to the panel softens that edge again.

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
)

// linuxTraySide is the side of the picture a Linux tray hands the desktop, a size the committed icon
// holds so nothing is resampled.
const linuxTraySide = 64

// opaqueFrom is the least coverage a pixel keeps: at least half covered is drawn, less is not.
const opaqueFrom = math.MaxUint8/2 + 1

// trayPicture answers the PNG frame with every pixel made fully opaque or fully clear.
func trayPicture(frame []byte) ([]byte, error) {
	decoded, err := png.Decode(bytes.NewReader(frame))
	if err != nil {
		return nil, fmt.Errorf("reading the tray's picture: %w", err)
	}
	picture := image.NewNRGBA(decoded.Bounds())
	draw.Draw(picture, picture.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	for alpha := len(picture.Pix) - 1; alpha >= 0; alpha -= 4 {
		if picture.Pix[alpha] >= opaqueFrom {
			picture.Pix[alpha] = math.MaxUint8
		} else {
			picture.Pix[alpha] = 0
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, picture); err != nil {
		return nil, fmt.Errorf("writing the tray's picture: %w", err)
	}
	return out.Bytes(), nil
}
