# openpnp-tools

[![CI](https://github.com/laenzlinger/openpnp-tools/actions/workflows/ci.yml/badge.svg)](https://github.com/laenzlinger/openpnp-tools/actions/workflows/ci.yml)

CLI toolkit for managing [OpenPnP](https://openpnp.org/) pick-and-place jobs.

Bridges the gap between KiCad PCB design and OpenPnP machine operation — generating
board files, assigning feeders, and visualizing the setup.

## Data flow

```
KiCad Schematic                KiCad PCB
      │                             │
      ├─ sch export bom             ├─ pcb export pos
      │  (Value,Footprint,IPN)      │  (Ref,Value,Footprint,X,Y,Rot,Side)
      ▼                             ▼
  pnp/bom.csv              position CSV
      │                             │
      │                             ▼
      │                     ┌──────────────┐
      │                     │   generate   │──▶ pnp/board.xml + pnp/board.pos
      │                     └──────────────┘
      │                             │
      │                             ▼
      │                     ┌──────────────┐
      │                     │ ensure-parts │──▶ parts.xml, packages.xml
      │                     └──────────────┘
      │
      │   pnp/feeders.csv ─▶┌──────────────┐
      │                      │    assign    │──▶ machine.xml (feeder slots)
      │                      └──────────────┘
      │
      │   pnp/job.xml ─────▶┌──────────────┐
      └────────────────────▶│     map      │──▶ pnp/feeder-map.html
                             └──────────────┘

Shared config (read by generate, map, ensure-parts):
  ~/.openpnp2/package-map.csv   Footprint → package mapping + tape metadata
  ~/.openpnp2/machine.xml       Feeder positions, bed dimensions
```

## Commands

### `generate` — KiCad → OpenPnP board

Reads a KiCad position CSV and generates an OpenPnP board XML with remapped
package names. Fiducials are auto-detected.

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
package-id and heights from the package map. Also sets `tape-specification`
and `compatible-nozzle-tip-ids` on packages in `packages.xml`.

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

### `reset-feeders` — prepare for version control

Resets all strip feeder `part-id` to `CALIBRATION-DUMMY` and zeroes `feed-count`.
Leaves positions, calibration, tape-type, rotation, and pitch untouched.

```bash
openpnp-tools reset-feeders                              # reset ~/.openpnp2/machine.xml
openpnp-tools reset-feeders --machine path/to/machine.xml  # reset a copy
```

Used by [openpnp-config](https://github.com/laenzlinger/openpnp-config) to strip
project-specific state before committing the base machine config.

### `map` — interactive feeder visualization

Generates a self-contained HTML page showing feeder positions, job parts,
missing feeders, and board outlines.

With `--bom`, adds an IPN column for cross-referencing against inventory
management (e.g. InvenTree).

```bash
# Basic
openpnp-tools map -o pnp/feeder-map.html pnp/myboard.job.xml

# With IPN from KiCad BOM
kicad-cli sch export bom --fields "Value,Footprint,IPN" \
  --labels "Value,Footprint,IPN" --group-by "Value,Footprint,IPN" \
  --exclude-dnp -o pnp/bom.csv myboard.kicad_sch
openpnp-tools map --bom pnp/bom.csv -o pnp/feeder-map.html pnp/myboard.job.xml
```

### `config` — manage base machine configuration

Subcommands for syncing the base config between a git repo
([openpnp-config](https://github.com/laenzlinger/openpnp-config)) and the
live `~/.openpnp2` directory. All subcommands share a `--config-dir` flag
(default: `~/.openpnp2`).

```bash
openpnp-tools config backup                          # snapshot ~/.openpnp2
openpnp-tools config apply --from /path/to/repo      # backup + copy repo → ~/.openpnp2
openpnp-tools config pull --to /path/to/repo         # copy ~/.openpnp2 → repo + reset feeders
openpnp-tools config status --repo /path/to/repo     # process check + per-file drift
```

- **backup** — copies all config XMLs to `~/.openpnp2/backups/<timestamp>/`,
  using the same directory and timestamp format as OpenPnP itself.
- **apply** — backs up first, then copies managed files from the repo into
  `~/.openpnp2`. Use `--no-backup` to skip. OpenPnP must be closed.
- **pull** — copies managed files from `~/.openpnp2` into the repo, then
  runs `reset-feeders` on the repo copy. Use `--no-reset` to skip.
  OpenPnP must be closed.
- **status** — reports whether OpenPnP is running and shows per-file drift
  between live config and the repo. Exits non-zero if OpenPnP is running.

## Global flags

These flags are available on all commands:

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--machine` | `~/.openpnp2/machine.xml` | Path to OpenPnP machine config |
| `--package-map` | `~/.openpnp2/openpnp-package-map.csv` | Path to footprint → package mapping CSV |

## Configuration

### Settings file (`~/.openpnp2/.openpnp-tools.yaml`)

Optional config file to avoid repeating flags. Looked up in the `--config-dir`
directory (default: `~/.openpnp2`).

```yaml
# Path to the openpnp-config git repo
repo-dir: /home/user/dev/openpnp-config
```

| Key | Used by | Description |
| --- | ------- | ----------- |
| `repo-dir` | `config apply`, `config pull`, `config status` | Default `--from`/`--to`/`--repo` directory |

With `repo-dir` set, you can run all config commands from any directory:

```bash
openpnp-tools config pull       # pulls to repo-dir
openpnp-tools config apply      # applies from repo-dir
openpnp-tools config status     # compares live vs repo-dir
```

CLI flags always override the config file.

### Package map (`~/.openpnp2/openpnp-package-map.csv`)

Single source of truth for package metadata, shared across all projects:

```csv
kicad_footprint,openpnp_package,height,tape_type,part_pitch,tape_width,nozzle_tip
C_0805_2012Metric,C_0805,0.9,WhitePaper,4,8,NT1
R_0805_2012Metric,R_0805,0.5,WhitePaper,4,8,NT1
SOT-23,SOT-23,1.1,BlackPlastic,8,8,NT1
SOIC-8_3.9x4.9mm_P1.27mm,SOIC-8,1.75,BlackPlastic,8,12,TIP16cbc9505c3e1916
```

| Column | Description |
| ------ | ----------- |
| `kicad_footprint` | KiCad footprint library name (exact match) |
| `openpnp_package` | Short OpenPnP package name |
| `height` | Component height in mm (set on parts in `parts.xml`) |
| `tape_type` | `WhitePaper`, `ClearPlastic`, or `BlackPlastic` (set on feeders in `machine.xml`) |
| `part_pitch` | Distance between parts on tape in mm (set on feeders in `machine.xml`) |
| `tape_width` | Tape width in mm: 8, 12, 16 (set as `tape-specification` on packages in `packages.xml`) |
| `nozzle_tip` | OpenPnP nozzle tip ID (set as `compatible-nozzle-tip-ids` on packages in `packages.xml`) |

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
make feeder-map   # visualize the setup (with IPN cross-reference)
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
	kicad-cli sch export bom --fields "Value,Footprint,IPN" \
		--labels "Value,Footprint,IPN" --group-by "Value,Footprint,IPN" \
		--exclude-dnp -o pnp/bom.csv $(PROJECT).kicad_sch
	openpnp-tools map --bom pnp/bom.csv \
		-o pnp/feeder-map.html pnp/$(PROJECT).job.xml
```

## Build

Requires [Go](https://go.dev/) 1.26+:

```bash
make build    # build binary
make test     # run tests
make lint     # run linter
```

## Acknowledgments

The interactive feeder map was inspired by
[psypnp](https://inductive-kickback.com/2020/10/psypnp-for-openpnp/) by
Pat Deegan — a collection of OpenPnP scripting utilities for feeder management
and job setup.

## License

GPL-3.0-or-later — aligned with [OpenPnP](https://github.com/openpnp/openpnp).
