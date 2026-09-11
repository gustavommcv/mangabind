// Command mangabind reorganizes a chapter-by-chapter manga download into one .cbz per volume.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	input := flag.String("input", "", "directory containing one folder per downloaded chapter")
	output := flag.String("output", "", "directory to write the generated .cbz files into")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: mangabind --input <dir> --output <dir>")
		os.Exit(2)
	}

	fmt.Fprintln(os.Stderr, "mangabind: pipeline not implemented yet")
	os.Exit(1)
}
