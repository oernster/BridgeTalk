// Package iconfile reads the pictures out of the application's .ico: the one committed icon file.
//
// Windows reads the icon out of the binary. Linux is handed pictures instead, for the tray icon and
// for the icons the flatpak installs, so each is taken from the .ico rather than from a second copy
// of the artwork (FR-814, FR-810). Every frame the icon generator writes is a PNG.
package iconfile

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// The .ico layout: a six byte header whose last two bytes count the frames, then a sixteen byte
// entry per frame holding its width, its height, its length and where its data starts.
const (
	headerSize   = 6
	countOffset  = 4
	entrySize    = 16
	lengthOffset = 8
	startOffset  = 12
	// fullSide is the side a width or height byte of zero stands for.
	fullSide = 256
)

// pngSignature opens every PNG file.
var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// ErrNoFrame means the .ico holds no square PNG frame of the side asked for.
var ErrNoFrame = errors.New("the icon holds no picture of that size")

// entry is one frame as the .ico describes it, checked against the file.
type entry struct {
	width, height int
	data          []byte
}

// Frame answers the square PNG picture of the given side inside an .ico. Nothing the file says
// about itself is trusted. An entry or a frame running past the end of the file is refused; so is a
// frame that is not a PNG.
func Frame(ico []byte, side int) ([]byte, error) {
	entries, err := entries(ico)
	if err != nil {
		return nil, err
	}
	for _, found := range entries {
		if found.width == side && found.height == side {
			return found.data, nil
		}
	}
	return nil, ErrNoFrame
}

// Sides answers the side of every square frame in an .ico, in the order the file holds them.
func Sides(ico []byte) ([]int, error) {
	entries, err := entries(ico)
	if err != nil {
		return nil, err
	}
	var sides []int
	for _, found := range entries {
		if found.width == found.height {
			sides = append(sides, found.width)
		}
	}
	return sides, nil
}

// entries reads and checks every frame an .ico describes.
func entries(ico []byte) ([]entry, error) {
	if len(ico) < headerSize {
		return nil, fmt.Errorf("the icon is %d bytes, too short to be one", len(ico))
	}
	count := int(binary.LittleEndian.Uint16(ico[countOffset:]))
	out := make([]entry, 0, count)
	for index := range count {
		at := headerSize + index*entrySize
		if at+entrySize > len(ico) {
			return nil, fmt.Errorf("the icon's entry %d runs past its end", index)
		}
		length := int64(binary.LittleEndian.Uint32(ico[at+lengthOffset:]))
		start := int64(binary.LittleEndian.Uint32(ico[at+startOffset:]))
		if start+length > int64(len(ico)) {
			return nil, fmt.Errorf("the icon's picture %d runs past its end", index)
		}
		data := ico[start : start+length]
		if !bytes.HasPrefix(data, pngSignature) {
			return nil, fmt.Errorf("the icon's picture %d is not a PNG", index)
		}
		out = append(out, entry{width: sideOf(ico[at]), height: sideOf(ico[at+1]), data: data})
	}
	return out, nil
}

// sideOf reads a width or height byte, where zero stands for the full side.
func sideOf(stored byte) int {
	if stored == 0 {
		return fullSide
	}
	return int(stored)
}
