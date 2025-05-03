package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func printUsage() {

	fmt.Println("Usage: map2bbc [flags] <mapfile> <outputfile>")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func exportMap(mapfile string, outfile string, packed bool) error {

	worldBytes, err := os.ReadFile(mapfile)

	if err != nil {

		return err
	}

	var world World
	err = json.Unmarshal(worldBytes, &world)

	if err != nil {

		return err
	}

	// Now loop over the world and write out the level data

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
