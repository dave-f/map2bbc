package main

import (
	"flag"
	"fmt"
)

func printUsage() {

	fmt.Println("Usage: map2bbc [flags] <mapfile> <outputfile>")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {

	packed := flag.Bool("p", false, "Output in Mountain Panic packed format")

	flag.Parse()

	if flag.NArg() != 2 {

		printUsage()
		return
	}

	if *packed {

		err := panicExport(flag.Arg(0), flag.Arg(1))

		if err != nil {

			fmt.Printf("Error exporting: %v\n", err)
		}
	} else {

		fmt.Println("Currently, only Mountain Panic format is supported")
	}

}
