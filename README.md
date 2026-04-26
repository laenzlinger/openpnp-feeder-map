# openpnp-tools

[![CI](https://github.com/laenzlinger/openpnp-tools/actions/workflows/ci.yml/badge.svg)](https://github.com/laenzlinger/openpnp-tools/actions/workflows/ci.yml)

CLI toolkit for managing [OpenPnP](https://openpnp.org/) pick-and-place jobs.

Bridges the gap between KiCad PCB design and OpenPnP machine operation — generating
board files, assigning feeders, and visualizing the setup.

## Commands

### `generate` — KiCad → OpenPnP board

Reads a KiCad position CSV and generates an OpenPnP board XML with remapped
package names. Fiducials are auto-detected. Missing parts are created in `parts.xml`.

```bash
# KiCad 10+ (writes to file)
kicad-cli pcb export pos --format csv --side both --units mm --smd-only --exclude-dnp board.kicad_pcb
openpnp-tools generate -o pnp/ -n myboard board.csv

# KiCad ≤9 (writes to stdout)
kicad-cli pcb export pos --format csv --side both --units mm --smd-only --exclude-dnp board.kicad_pcb \
  | openpnp-tools generate -o pnp/ -n myboard
```

Outputs:
- `pnp/myboard.board.xml` — OpenPnP board with placements
- `pnp/myboard.pos` — remapped position file

### `ensure-parts` — create missing parts

Ensures all parts referenced by a board exist in `parts.xml` with correct
package-id and heights from the package map.

```bash
openpnp-tools ensure-parts pnp/myboard.board.xml
```

### `assign` — load feeder configuration

Assigns parts to feeder slots in `machine.xml` based on a project's `feeders.csv`.
Also sets tape-type and part-pitch from the package map.

```bash
openpnp-tools assign pnp/feeders.csv
openpnp-tools assign --reset-unused pnp/feeders.csv  # reset unassigned feeders
openpnp-tools assign --dry-run pnp/feeders.csv       # preview changes
```

### `map` — interactive feeder visualization

Generates a self-contained HTML page showing feeder positions, job parts,
missing feeders, and board outlines.

```bash
openpnp-tools map -o pnp/feeder-map.html pnp/myboard.job.xml
```

## Configuration

### Package map (`~/.openpnp2/openpnp-package-map.csv`)

Single source of truth for package metadata, shared across all projects:

```csv
kicad_footprint,openpnp_package,height,tape_type,part_pitch,tape_width
C_0805_2012Metric,C_0805,0.9,WhitePaper,4,8
R_0805_2012Metric,R_0805,0.5,WhitePaper,4,8
SOT-23,SOT-23,1.1,ClearPlastic,8,8
SOIC-8_3.9x4.9mm_P1.27mm,SOIC-8,1.75,ClearPlastic,12,12
```

| Column | Description |
| ------ | ----------- |
| `kicad_footprint` | KiCad footprint library name (exact match) |
| `openpnp_package` | Short OpenPnP package name |
| `height` | Component height in mm (set on parts in `parts.xml`) |
| `tape_type` | `WhitePaper`, `ClearPlastic`, or `BlackPlastic` (set on feeders in `machine.xml`) |
| `part_pitch` | Distance between parts on tape in mm (set on feeders in `machine.xml`) |
| `tape_width` | Tape width in mm: 8, 12, 16 (set as `tape-specification` on packages in `packages.xml`) |

Lines starting with `#` are comments. Rows with empty tape columns are hand-place
components (connectors, switches) — they are mapped but get no feeder metadata.

### Feeder allocation (`pnp/feeders.csv`)

Per-project file mapping feeder slots to parts:

```csv
feeder,part
LV08-01,C_0805-100n
RV08-02,SOT-23-2N7002
RH12-01,SOT-223-NCP1117-3.3_SOT223
```

| Column | Description |
| ------ | ----------- |
| `feeder` | Feeder slot name: `{L/R}{V/H}{width}-{slot}` zero-padded (e.g. `LV08-01`, `RH12-03`) |
| `part` | Part ID: `{Package}-{Value}` matching the board XML (e.g. `C_0805-100n`) |

Lines starting with `#` are comments.

Feeder naming convention:
- `L`/`R` = left/right side of machine bed
- `V`/`H` = vorne (front) / hinten (back)
- Width = tape width in mm (08, 12, 16)
- Slot = position number, zero-padded

## Typical workflow

### First-time project setup

```bash
make pnp          # generate board.xml + ensure parts
make feeders      # assign parts to feeder slots
make feeder-map   # visualize the setup
```

### After PCB revision

```bash
make pnp          # regenerate with new placements
make feeders      # re-assign (new parts flagged as not_found)
# update feeders.csv for new parts, then:
make feeders      # assign new parts
```

### Switching between projects

```bash
cd other-project/hardware
make feeders      # reassigns all feeder slots for this project
# swap tape strips to match, then run OpenPnP
```

## Makefile integration

```makefile
pnp:
	kicad-cli pcb export pos --format csv --side both --units mm \
		--smd-only --exclude-dnp board.kicad_pcb
	openpnp-tools generate -o pnp -n $(PROJECT) board.csv
	openpnp-tools ensure-parts pnp/$(PROJECT).board.xml

feeders:
	openpnp-tools assign pnp/feeders.csv

feeder-map:
	openpnp-tools map -o pnp/feeder-map.html pnp/$(PROJECT).job.xml
```

## Build

Requires [Go](https://go.dev/) 1.26+:

```bash
make build    # build binary
make test     # run tests
make lint     # run linter
```

## License

GPL-3.0-or-later — aligned with [OpenPnP](https://github.com/openpnp/openpnp).
