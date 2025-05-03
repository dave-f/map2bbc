package main

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

func printUsage() {

	fmt.Println("Usage: map2bbc [flags] <mapfile> <outputfile>")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func exportMap(mapfile string, outfile string, packed bool) error {

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

	// write NUM_SCREENS = len(world.Maps)
	fmt.Println("NUM_SCREENS =", len(world.Maps))

	// Now loop over the world and write out the level data
	for _, v := range world.Maps {

		mapBytes, err := os.ReadFile(worldPath + "/" + v.Filename)

		if err != nil {

			return err
		}

		var tilemap TileMap
		err = xml.Unmarshal(mapBytes, &tilemap)

		if err != nil {

			return err
		}

		// The CSV reader expects no ',' at the end of a line, so remove those
		trimmed := strings.ReplaceAll(tilemap.Data, ",\n", "\n")
		reader := csv.NewReader(strings.NewReader(trimmed))
		records, err := reader.ReadAll()

		for _, i := range records {

			outputStr := "EQUB "

			for _, j := range i {

				thisByte, err := strconv.Atoi(j)

				if err != nil {

					return err
				}

				outputStr += fmt.Sprintf("&%02X ", thisByte)
			}

			fmt.Println(outputStr)
		}

		fmt.Println()

	}

	return nil
}

func main() {

	packed := flag.Bool("p", false, "Output in Mountain Panic packed format")

	flag.Parse()

	if flag.NArg() != 2 {

		printUsage()
		return
	}

	err := exportMap(flag.Arg(0), flag.Arg(1), *packed)

	if err != nil {

		println("Error exporting:", err.Error())
	}
}
