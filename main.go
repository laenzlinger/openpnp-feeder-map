// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "github.com/laenzlinger/openpnp-feeder-map/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
