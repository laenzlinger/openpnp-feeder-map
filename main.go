// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/laenzlinger/openpnp-feeder-map/internal/feedermap"
	"github.com/laenzlinger/openpnp-feeder-map/internal/openpnp"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultMachine := filepath.Join(home, ".openpnp2", "machine.xml")

	machineFlag := flag.String("machine", defaultMachine, "path to machine.xml")
	outputFlag := flag.String("output", "feeder-map.html", "output HTML file path")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: openpnp-feeder-map [flags] <job.xml>\n\nGenerates an interactive HTML feeder map for an OpenPnP job.\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	jobPath := flag.Arg(0)

	// Parse inputs
	jobParts, boards, err := openpnp.LoadJobParts(jobPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading job: %v\n", err)
		os.Exit(1)
	}

	machine, err := openpnp.ParseMachine(*machineFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading machine: %v\n", err)
		os.Exit(1)
	}

	// Build feeder map
	data := feedermap.Build(jobParts, boards, machine)
	data.JobFile = filepath.Base(jobPath)

	// Render HTML
	f, err := os.Create(*outputFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	if err := feedermap.Render(f, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Feeder map: %d feeders, %d missing parts, %d unused → %s\n",
		len(data.Feeders), len(data.MissingParts), len(data.UnusedFeeders), *outputFlag)
}
