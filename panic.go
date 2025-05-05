package main

import "errors"

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

func packLine(b []byte) ([]byte, error) {

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

	if rleCnt > 1 {

		idx++
	}

	// If we have an RLE count of 1 or 2, just output the bytes
	if rleCnt < 3 {

		if currentByte != 0 {

			for range rleCnt {

				r = append(r, currentByte)
			}
		}

		if rleCnt == 1 {

			idx++
		}
	} else {

		// output 0xf | rleCnt plus the byte itself
		r = append(r, byte(0xf0|rleCnt))
		r = append(r, currentByte)
	}

	if idx < 8 {

		goto loop
	}

	return r, nil
}
