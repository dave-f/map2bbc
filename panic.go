package main

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const _bitClimbable = 0x80
const _bitCollidable = 0x40
const _bitHookable = 0x20

type PanicCoord struct {
	X int
	Y int
}

type PanicScreen struct {
	Source      string // The file
	Tileset     int    // 0,1,2 or 3
	Name        string
	Effect      string // e.g. "stars" or somesuch
	ExitUp      bool
	ExitDown    bool
	ExitRight   bool
	ExitLeft    bool
	TileIndexes [][]int // TODO
	// TODO Aliens and items
}

const WorldMaxSize = 256
const WorldOrigin = WorldMaxSize / 2

var PanicWorld [][]PanicScreen
var PanicScreens map[string]PanicCoord // TODO change this to a normal slice I think so we can write out in an expected order

// Parse out an individual screen
func panicParseScreen(filename string) (PanicScreen, error) {

	// get screenname, exits, items, aliens etc
	var r PanicScreen

	bytes, err := os.ReadFile(filename)

	if err != nil {

		return r, err
	}

	var m TileMap
	err = xml.Unmarshal(bytes, &m)

	if err != nil {

		return r, err
	}

	// Properties.  For now, we just get the name, and the exit states.  Note that the exit values
	// themselves currently don't matter as just their presence is enough to trigger an exit here.
	for _, v := range m.Properties {

		if v.Name == "name" {
			r.Name = v.Value
		} else if v.Name == "exitUp" {
			r.ExitUp = true
		} else if v.Name == "exitDown" {
			r.ExitDown = true
		} else if v.Name == "exitLeft" {
			r.ExitLeft = true
		} else if v.Name == "exitRight" {
			r.ExitRight = true
		}
	}

	_, r.Source = filepath.Split(filename)

	return r, nil
}

// Parse the world into a 2D slice with the start screen in the centre
// Also create the slice containing the levels, so we can write them out in an expected order to leveldata.asm
func panicParseWorld(path string, w World) error {

	PanicWorld = make([][]PanicScreen, WorldMaxSize)
	PanicScreens = make(map[string]PanicCoord)

	for i := range PanicWorld {

		PanicWorld[i] = make([]PanicScreen, WorldMaxSize)
	}

	for _, v := range w.Maps {

		if v.Width != 128 || v.Height != 192 {

			return errors.New("bad dimensions")
		}

		ourX := (v.WorldX / 128) + WorldOrigin
		ourY := (v.WorldY / 192) + WorldOrigin

		ps, err := panicParseScreen(path + "/" + v.Filename)

		if err != nil {

			return err
		}

		PanicWorld[ourX][ourY] = ps

		_, exists := PanicScreens[v.Filename]

		if exists {

			return errors.New("screen used more than once")
		}

		PanicScreens[v.Filename] = PanicCoord{ourX, ourY}
	}

	return nil
}

func panicGetScreen(filename string) (PanicScreen, error) {

	coords, exist := PanicScreens[filename]

	if !exist {

		return PanicScreen{}, errors.New("screen not found")
	}

	return PanicWorld[coords.X][coords.Y], nil
}

// Get the flags for this tile type
// This needs some work because some tiles - beyond the default - have flags like collidable AND climbable
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

// Pack a row of screen bytes.
// It is expected the slice passed in has 8 bytes in it.
//
// The return slice is variable length, and in the "Mountain Panic" format:
// First byte is a control byte, with each bit set meaning a tile is present in that slot on the row.
// Following this, either run length byte packing:
//
//	0xf0 - Not yet done, but if needed : rest of the line is the same repeated tile, which follows this byte
//	0xfn - The next tile is repeated n times (n = 0-15)
//
// Or a normal tile index:
//
//	0x00 - 0x1f
func panicPackLine(b []byte, includeFlags bool) ([]byte, error) {

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

// Write the leveldata.asm header
func panicWriteHeader(f *os.File) {

	fmt.Fprintln(f, "_effectStars = &10")
	fmt.Fprintln(f, "_effectPaletteChange = &20 ; change to red for hell")
	fmt.Fprintln(f, "_effectPaletteChange2 = &40 ; change to magenta for catacombs area")
	fmt.Fprintln(f, "_effectGems = &80 ; read inv bits, plot red/green boxes for eyes on statues")
	fmt.Fprintln(f, "_effectDark = &08 ; dark unless have torch")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "SCREEN_FLAGS_ITEM_PRESENT = &04")
}

// Write the leveldata.asm footer
func panicWriteFooter(f *os.File) {

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

// Export the map in Mountain Panic leveldata.asm format
func panicExport(mapfile string, outfile string) error {

	worldPath := path.Dir(mapfile)
	worldBytes, err := os.ReadFile(mapfile)

	if err != nil {

		return err
	}

	var world World
	err = json.Unmarshal(worldBytes, &world)

	if err != nil {

		return err
	}

	if len(world.Maps) == 0 {

		return errors.New("no maps")
	}

	f, err := os.Create(outfile)

	if err != nil {

		return err
	}

	defer f.Close()

	fmt.Fprintln(f, "; Auto-generated by map2bbc on", time.Now().Format("Mon, 2 Jan at 15:04"))
	fmt.Fprintln(f)
	fmt.Fprintln(f, "NUM_SCREENS =", len(world.Maps))

	// Create our world structure so we can get the screen exits etc
	err = panicParseWorld(worldPath, world)

	if err != nil {

		return err
	}

	fmt.Fprintln(f)
	panicWriteHeader(f)
	fmt.Fprintln(f)
	fmt.Fprintln(f, ".mapData:")

	for i, v := range world.Maps {

		s, err := panicGetScreen(v.Filename)

		if err != nil {

			return err
		}

		// TODO: Add exits etc here
		fmt.Fprintf(f, "  ; %s\n", s.Name)

		fmt.Fprintf(f, "  ; Screen %d\n", i)
		fmt.Fprintln(f, "  EQUB &00")
		fmt.Fprintln(f, "  EQUB &00")
		fmt.Fprintln(f, "  EQUB &00")
		fmt.Fprintln(f, "  EQUB &00,&00, &00, &00")
		fmt.Fprintln(f, "  EQUB &00")
	}

	fmt.Fprintln(f)
	fmt.Fprintln(f, ".screenTable:")
	for i := range len(world.Maps) {

		fmt.Fprintf(f, "  EQUW screen%dData\n", i)
	}
	fmt.Fprintln(f)

	// Now loop over the world and write out the level data
	for i, v := range world.Maps {

		mapBytes, err := os.ReadFile(worldPath + "/" + v.Filename)

		if err != nil {

			return err
		}

		var tilemap TileMap
		err = xml.Unmarshal(mapBytes, &tilemap)

		if err != nil {

			return err
		}

		fmt.Fprintf(f, ".screen%dData:\n", i)

		// The CSV reader expects no ',' at the end of a line, so remove those
		trimmed := strings.ReplaceAll(tilemap.Data, ",\n", "\n")
		reader := csv.NewReader(strings.NewReader(trimmed))
		records, err := reader.ReadAll()

		for _, i := range records {

			rowBytes := make([]byte, 0)

			for _, j := range i {

				thisByte, err := strconv.Atoi(j)

				if err != nil {

					return err
				}

				rowBytes = append(rowBytes, byte(thisByte))
			}

			outputStr := "  EQUB "

			packedRowBytes, err := panicPackLine(rowBytes, true)

			if err != nil {

				return err
			}

			for _, k := range packedRowBytes {

				outputStr += fmt.Sprintf("&%02X,", k)
			}

			outputStr, _ = strings.CutSuffix(outputStr, ",")

			fmt.Fprintln(f, outputStr)
		}

		fmt.Fprintln(f)
	}

	panicWriteFooter(f)

	return nil
}
