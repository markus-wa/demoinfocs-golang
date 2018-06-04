package examples

import (
	"flag"
	"log"
	"os"
)

// DemoPathFromArgs returns the value of the -demo command line flag.
func DemoPathFromArgs() string {
	fl := new(flag.FlagSet)

	demPathPtr := fl.String("demo", "", "Demo file `path`")

	err := fl.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	demPath := *demPathPtr

	return demPath
}
