package main

import (
	"fmt"
	"os"

	// The VMI framework:
	"github.com/bgp59/victoriametrics-importer/vmi"

	// The specific metrics generators:
	"github.com/bgp59/victoriametrics-importer/refvmi/refvmi"
)

const (
	DEFAULT_INSTANCE = "refvmi"
)

// Create the main log:
var mainLog = vmi.NewCompLogger(DEFAULT_INSTANCE)

// Customize the VMI framework for this particular instance. This should be done
// before invoking `vmi.Run', so it best to do it via `init()'.
func init() {
	// Add the prefix to strip when logging source file path for messages from
	// this module, based on the location of this file:
	vmi.AddCallerSrcPathPrefixToLogger(0) // this file is at the root

	// Default importer instance:
	vmi.SetDefaultInstance(DEFAULT_INSTANCE)

	// Default config file:
	vmi.SetDefaultConfigFile(fmt.Sprintf("%s-config.yaml", DEFAULT_INSTANCE))

	// The build info for this importer instance, based on auto-generated
	// buildinfo.go:
	vmi.UpdateBuildInfo(Version, GitInfo)
}

func main() {
	mainLog.Info("Start")
	// Invoke the runner with the default config as an argument. The former will
	// load the `generators' section of the config file into the later. The
	// metrics generators have registered their task builders with the VMI
	// framework via `init()' and the runner will invoke those, using the loaded
	// config as an argument, to retrieve the list of tasks to schedule. This is
	// how it all comes together, folks.
	os.Exit(vmi.Run(refvmi.DefaultRefvmiConfig()))
}
