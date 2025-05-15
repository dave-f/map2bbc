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

type PanicScreen struct {
	Occupied    bool
	GridX       int
	GridY       int
	Source      string // The source .tmx file
	Tileset     int    // 0,1,2 or 3
	Name        string // Screen name
	Effect      string // e.g. "stars" or somesuch
	ExitUp      bool
	ExitDown    bool
	ExitRight   bool
	ExitLeft    bool
	TileIndexes [][]byte
	// TODO Items
	// TODO Aliens
}

const WorldMaxSize = 256
const WorldOrigin = WorldMaxSize / 2

var PanicWorld [][]PanicScreen

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

	r.TileIndexes = make([][]byte, 0)

	for _ = range 12 {

		r.TileIndexes = append(r.TileIndexes, make([]byte, 0))
	}

	// Get the map
	tileLayerData, err := m.GetMapBaseLayer()

	if err != nil {

		return r, err
	}

	// The CSV reader expects no ',' at the end of a line, so remove those
	trimmed := strings.ReplaceAll(tileLayerData, ",\n", "\n")
	reader := csv.NewReader(strings.NewReader(trimmed))
	records, err := reader.ReadAll()
	rowCount := 0

	for _, i := range records {

		// rowBytes := make([]byte, 0)

		for _, j := range i {

			thisByte, err := strconv.Atoi(j)

			if err != nil {

				return r, err
			}

			r.TileIndexes[rowCount] = append(r.TileIndexes[rowCount], byte(thisByte))
		}

		rowCount = rowCount + 1

	}

	return r, nil
}

// Parse the world into a 2D slice with the start screen in the centre
// Also create the slice containing the levels, so we can write them out in an expected order to leveldata.asm
func panicParseWorld(path string, w World) error {

	PanicWorld = make([][]PanicScreen, WorldMaxSize)

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

		ps.Occupied = true
		ps.GridX = ourX
		ps.GridY = ourY

		PanicWorld[ourX][ourY] = ps
	}

	return nil
}

func panicGetTotalScreens() int {

	screenCount := 0

	for i := range len(PanicWorld) {

		for j := range len(PanicWorld[i]) {

			if PanicWorld[i][j].Occupied {

				screenCount = screenCount + 1
			}
		}
	}

	return screenCount
}

func panicGetScreen(screenNo int) (PanicScreen, error) {

	screenCount := 0

	for i := range len(PanicWorld) {

		for j := range len(PanicWorld[i]) {

			if PanicWorld[i][j].Occupied {

				if screenCount == screenNo {

					return PanicWorld[i][j], nil
				}

				screenCount = screenCount + 1
			}
		}
	}

	return PanicScreen{}, errors.New("screen not found")
}

func panicGetScreenNoFromGridPos(gridX int, gridY int) (int, error) {

	screenCount := 0

	for i := range len(PanicWorld) {

		for j := range len(PanicWorld[i]) {

			if PanicWorld[i][j].Occupied {

				if i == gridX && j == gridY {

					return screenCount, nil
				} else {

					screenCount++
				}
			}
		}
	}

	return -1, errors.New("screen not found")
}

// Get the flags for this tile type
// This needs some work because some tiles - beyond the default - have flags like collidable AND climbable
func panicGetFlagsForTile(tileID byte) byte {

	// Take 1 off, then or the flags in
	tileID = tileID - 1
	flags := byte(0)

	// Just for now, these will eventually be read out of the files above
	if tileID == 0 || tileID == 19 {

		flags = _bitClimbable
	} else if tileID == 2 || tileID == 8 || tileID == 17 {

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

	fmt.Fprintln(f, "; Auto-generated by map2bbc on", time.Now().Format("Mon, 2 Jan at 15:04"))
	fmt.Fprintln(f)
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

	fmt.Fprintln(f)
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

// TODO: the screens are 1+ at the moment to account for the title page
func panicGetExitBytes(s PanicScreen) ([]byte, error) {

	bytes := make([]byte, 4)

	if s.ExitUp {

		newScreen, err := panicGetScreenNoFromGridPos(s.GridX, s.GridY-1)

		if err != nil {

			return bytes, err
		}

		bytes[0] = byte(newScreen) + 1
	}

	if s.ExitDown {

		newScreen, err := panicGetScreenNoFromGridPos(s.GridX, s.GridY+1)

		if err != nil {

			return bytes, err
		}

		bytes[1] = byte(newScreen) + 1
	}

	if s.ExitRight {

		newScreen, err := panicGetScreenNoFromGridPos(s.GridX+1, s.GridY)

		if err != nil {

			return bytes, err
		}

		bytes[2] = byte(newScreen) + 1
	}

	if s.ExitLeft {

		newScreen, err := panicGetScreenNoFromGridPos(s.GridX-1, s.GridY)

		if err != nil {

			return bytes, err
		}

		bytes[3] = byte(newScreen) + 1
	}

	return bytes, nil
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

	err = panicParseWorld(worldPath, world)

	if err != nil {

		return err
	}

	panicWriteHeader(f)

	totalScreens := panicGetTotalScreens()

	fmt.Fprintln(f, "NUM_SCREENS =", totalScreens+1)
	fmt.Fprintln(f)
	fmt.Fprintln(f, ".mapData:")

	fmt.Fprintln(f, "  ; Screen 00 : title page")
	fmt.Fprintln(f, "  EQUB &00")
	fmt.Fprintln(f, "  EQUB &00")
	fmt.Fprintln(f, "  EQUB &00")
	fmt.Fprintln(f, "  EQUB &00,&00,&00,&00")
	fmt.Fprintln(f, "  EQUB &00")

	for i := range totalScreens {

		s, err := panicGetScreen(i)

		if err != nil {

			return err
		}

		eb, err := panicGetExitBytes(s)

		if err != nil {

			return err
		}

		si, err := panicGetScreenNoFromGridPos(s.GridX, s.GridY)

		if err != nil {

			return err
		}

		fmt.Fprintf(f, "  ; Screen %02d : %s\n", 1+i, s.Name)
		fmt.Fprintln(f, "  EQUB &00")
		fmt.Fprintf(f, "  EQUB &%02X\n", si)
		fmt.Fprintln(f, "  EQUB &00")
		fmt.Fprintf(f, "  EQUB &%02X,&%02X,&%02X,&%02X\n", eb[0], eb[1], eb[2], eb[3])
		fmt.Fprintln(f, "  EQUB &00")
	}

	fmt.Fprintln(f)
	fmt.Fprintln(f, ".screenTable:")

	for i := range totalScreens {

		fmt.Fprintf(f, "  EQUW screen%dData\n", i)
	}

	fmt.Fprintln(f)

	for i := range totalScreens {

		fmt.Fprintf(f, ".screen%dData:\n", i)

		s, err := panicGetScreen(i)

		if err != nil {

			return err
		}

		for _, v := range s.TileIndexes {

			outputStr := "  EQUB "
			packedRowBytes, err := panicPackLine(v, true)

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
