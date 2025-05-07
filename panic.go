package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
)

// map of TILEID to int (flags) ?

const _bitClimbable = 0x80
const _bitCollidable = 0x40
const _bitHookable = 0x20

func panicParseTileFlags(filename string) error {

	fileBytes, err := os.ReadFile(filename)

	if err != nil {

		return err
	}

	var ts TileSet
	err = xml.Unmarshal(fileBytes, &ts)

	if err != nil {

		return err
	}

	return nil
}

// Get the flags for this tile type

func panicGetFlagsForTile(tileID byte) byte {

	// Take 1 off, then or the flags in
	tileID = tileID - 1
	flags := byte(0)

	// Just for now, these will eventually be read out of the files above
	if tileID == 0 {

		flags = _bitClimbable
	} else if tileID == 2 {

		flags = _bitCollidable
	}

	return tileID | flags
}

// Pack a row of screen bytes
// It is expected the slice passed in has 8 bytes in it.
//
// The return slice is a variable length packed in the "Mountain Panic" format:
// First byte is a control byte, with each bit set meaning a tile is present in that slot on the row.
// Following this, either a run length byte:
//                 0xf0 - TODO (if needed) Rest of the line is the same repeated tile, which follows this byte
//                 0xfn - The next tile is repeated n times (n = 0-15)
// Or a normal tile index:
//                 0x00 - 0x1f

func packLine(b []byte, includeFlags bool) ([]byte, error) {

	if len(b) != 8 {

		return nil, errors.New("bad slice")
	}

	var bitVal byte
	r := make([]byte, 0, 9)

	for b, i := range b {

		if i != 0 {

			bitVal |= 1 << (7 - b)
		}
	}

	if bitVal == 0 {

		r = append(r, 0) // Empty row, just return [0]
		return r, nil
	}

	r = append(r, bitVal) // One bit set for each tile (i.e. 8 tiles in a row)

	// Now do the RLE packing
	idx := 0

loop:
	currentByte := b[idx]
	rleCnt := 1

	for idx < 7 && currentByte == b[idx+1] && currentByte != 0 {

		rleCnt++
		idx++
	}

	idx++

	// If we have an RLE count of 1 or 2, just output the bytes
	if rleCnt < 3 {

		if currentByte != 0 {

			for range rleCnt {

				if includeFlags {

					currentByte = panicGetFlagsForTile(currentByte)
				}

				r = append(r, currentByte)
			}
		}
	} else {

		if currentByte == 0 {

			return nil, errors.New("unexpected byte")
		}

		// output 0xf | rleCnt plus the byte itself
		r = append(r, byte(0xf0|rleCnt))

		if includeFlags {

			currentByte = panicGetFlagsForTile(currentByte)
		}

		r = append(r, currentByte)
	}

	if idx < 8 {

		goto loop
	}

	return r, nil
}

func writePanicHeader(f *os.File) {

	fmt.Fprintln(f, "_effectStars = &10")
	fmt.Fprintln(f, "_effectPaletteChange = &20 ; change to red for hell")
	fmt.Fprintln(f, "_effectPaletteChange2 = &40 ; change to magenta for catacombs area")
	fmt.Fprintln(f, "_effectGems = &80 ; read inv bits, plot red/green boxes for eyes on statues")
	fmt.Fprintln(f, "_effectDark = &08 ; dark unless have torch")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "SCREEN_FLAGS_ITEM_PRESENT = &04")
}

func writePanicFooter(f *os.File) {

	fmt.Fprintln(f, "; These values are per screen and copied into the zp 'snowwindow' memory on a screen change")
	fmt.Fprintln(f, ".snowWindowValueTable:")
	fmt.Fprintln(f, "  EQUB 12,15,26,38 ; in the cave, poor lake, east tower, west tower")
	fmt.Fprintln(f, "  EQUB 0")
	fmt.Fprintln(f)
	fmt.Fprintln(f, ".snowWindowValues:")
	fmt.Fprintln(f, "  EQUB 16*4, 16*0, 16*7, 16*7 ; In the cave")
	fmt.Fprintln(f, "  EQUB 16*4, 16*0, 16*8, 16*5 ; Poor lake")
	fmt.Fprintln(f, "  EQUB 0, 16*0, 16*8, 16*5    ; Top of east tower")
	fmt.Fprintln(f, "  EQUB 0, 0, 128, (16*8)-8    ; Top of west tower")
	fmt.Fprintln(f)
	fmt.Fprintln(f, ".endLevelData:")
	fmt.Fprintln(f, "  PRINT \"Level data takes \", P%-mapData")
}
