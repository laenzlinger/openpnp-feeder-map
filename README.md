# openpnp-tools

[![CI](https://github.com/laenzlinger/openpnp-tools/actions/workflows/ci.yml/badge.svg)](https://github.com/laenzlinger/openpnp-tools/actions/workflows/ci.yml)

CLI toolkit for managing [OpenPnP](https://openpnp.org/) pick-and-place jobs.

Bridges the gap between KiCad PCB design and OpenPnP machine operation — generating
board files, assigning feeders, and visualizing the setup.

## Data flow

```
KiCad Schematic                 KiCad PCB
      │                              │
      ├─ sch export bom              ├─ pcb export pos
      │  (Value,Footprint,IPN)       │  (Ref,Value,Footprint,X,Y,Rot,Side)
      ▼                              ▼
  pnp/bom.csv               position CSV
      │                              │
      │                              ▼
      │                      ┌──────────────┐
      │                      │   generate   │──▶ pnp/board.xml + pnp/board.pos
      │                      └──────────────┘
      │                              │
      │                              ▼
      │                      ┌──────────────┐
      │                      │ ensure-parts │──▶ parts.xml, packages.xml
      │                      └──────────────┘
      │
      │   pnp/feeders.csv ─▶ ┌──────────────┐
      │                      │    assign    │──▶ machine.xml (feeder slots)
      │                      └──────────────┘
      │
      │   pnp/job.xml ─────▶ ┌──────────────┐
      └────────────────────▶ │     map      │──▶ pnp/feeder-map.html
                             └──────────────┘

Shared config (from config repo, resolved via repo-dir):
  openpnp-package-map.csv    Footprint → package mapping + tape metadata
  machine.xml                Feeder positions, bed dimensions
```

## Commands

### `generate` — KiCad → OpenPnP board

Reads a KiCad position CSV and generates an OpenPnP board XML with remapped
package names. Fiducials are auto-detected. Hand-place parts (packages with
no tape info in the package map) are automatically disabled.

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
Also sets tape-type, part-pitch, and rotation from the package map.

```bash
openpnp-tools assign pnp/feeders.csv
openpnp-tools assign --reset-unused pnp/feeders.csv  # reset unassigned feeders
openpnp-tools assign --dry-run pnp/feeders.csv       # preview changes
```

### `reset-feeders` — prepare for version control

Resets all strip feeder `part-id` to `CALIBRATION-DUMMY` and zeroes `feed-count`.
Leaves positions, calibration, tape-type, rotation, and pitch untouched.

```bash
openpnp-tools reset-feeders                                # reset ~/.openpnp2/machine.xml
openpnp-tools reset-feeders --machine path/to/machine.xml  # reset a copy
```

### `map` — interactive feeder visualization

Generates a self-contained HTML page showing feeder positions, job parts,
missing feeders, and board outlines.

With `--bom`, adds an IPN column for cross-referencing against inventory
management (e.g. InvenTree).

```bash
openpnp-tools map -o pnp/feeder-map.html pnp/myboard.job.xml

# With IPN from KiCad BOM
kicad-cli sch export bom --fields "Value,Footprint,IPN" \
  --labels "Value,Footprint,IPN" --group-by "Value,Footprint,IPN" \
  --exclude-dnp -o pnp/bom.csv myboard.kicad_sch
openpnp-tools map --bom pnp/bom.csv -o pnp/feeder-map.html pnp/myboard.job.xml
```

### `config` — manage base machine configuration

Syncs the base config between a git repo and the live `~/.openpnp2` directory.
With `repo-dir` set in the settings file, all commands work from any directory.

```bash
openpnp-tools config backup       # snapshot ~/.openpnp2
openpnp-tools config apply        # backup + copy repo → ~/.openpnp2
openpnp-tools config pull         # copy ~/.openpnp2 → repo + reset feeders
openpnp-tools config save         # pull + commit + push in one step
openpnp-tools config status       # process check + per-file drift
openpnp-tools config status -d    # include unified diff for modified files
```

- **backup** — copies all config XMLs to `~/.openpnp2/backups/<timestamp>/`,
  using the same format as OpenPnP itself.
- **apply** — backs up first, then copies managed files from the repo into
  `~/.openpnp2`. OpenPnP must be closed.
- **pull** — copies managed files from `~/.openpnp2` into the repo, then
  runs `reset-feeders` on the repo copy. OpenPnP must be closed.
- **save** — pull + reset + git commit + push. Use `-m` for a custom message.
- **status** — reports whether OpenPnP is running and shows per-file drift.
  Use `-d` to show the actual diff.

## Global flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--machine` | `~/.openpnp2/machine.xml` | Path to OpenPnP machine config |
| `--package-map` | `<repo-dir>/openpnp-package-map.csv` | Path to package map CSV (falls back to `~/.openpnp2/`) |

## Configuration

### Settings file (`~/.openpnp2/.openpnp-tools.yaml`)

Optional config file to avoid repeating flags. Looked up in the `--config-dir`
directory (default: `~/.openpnp2`).

```yaml
repo-dir: /home/user/dev/openpnp-config
```

| Key | Used by | Description |
| --- | ------- | ----------- |
| `repo-dir` | `config apply/pull/save/status`, `--package-map` | Path to the config git repo |

With `repo-dir` set, the package map is read directly from the repo and all
config commands work from any directory. CLI flags always override.

### Package map (`openpnp-package-map.csv`)

Single source of truth for package metadata, shared across all projects.
Lives in the config repo (resolved via `repo-dir`).

```csv
kicad_footprint,openpnp_package,height,tape_type,part_pitch,tape_width,nozzle_tip,rotation_offset
C_0805_2012Metric,C_0805,0.9,WhitePaper,4,8,NT1,90
R_0805_2012Metric,R_0805,0.5,WhitePaper,4,8,NT1,90
SOT-23,SOT-23,1.1,BlackPlastic,4,8,NT1,0
SOIC-8_3.9x4.9mm_P1.27mm,SOIC-8,1.75,BlackPlastic,8,12,TIP16cbc9505c3e1916,0
```

| Column | Description |
| ------ | ----------- |
| `kicad_footprint` | KiCad footprint library name (exact match) |
| `openpnp_package` | Short OpenPnP package name |
| `height` | Component height in mm |
| `tape_type` | `WhitePaper`, `ClearPlastic`, or `BlackPlastic` |
| `part_pitch` | Distance between parts on tape in mm |
| `tape_width` | Tape width in mm (8, 12, 16) |
| `nozzle_tip` | OpenPnP nozzle tip ID |
| `rotation_offset` | Degrees for "Rotation in Tape" in OpenPnP |

Lines starting with `#` are comments. Rows with empty tape columns are hand-place
components — they are mapped but get no feeder metadata, and `generate` marks
their placements as disabled.

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
| `feeder` | Feeder slot name (e.g. `LV08-01`, `RH12-03`) |
| `part` | Part ID: `{Package}-{Value}` matching the board XML |

## Typical workflow

### Before a job

```bash
make setup        # apply base config + assign feeders
# open OpenPnP, run the job
```

### After a job

```bash
# close OpenPnP
make save         # pull tuning + commit + push
```

### After PCB revision

```bash
make pnp          # regenerate board.xml + ensure parts
make setup        # apply + assign (update feeders.csv first if new parts)
```

## Makefile integration

```makefile
PROJECT = myboard

pnp:
	kicad-cli pcb export pos --format csv --side both --units mm \
		--smd-only --exclude-dnp $(PROJECT).kicad_pcb
	openpnp-tools generate -o pnp -n $(PROJECT) $(PROJECT).csv
	openpnp-tools ensure-parts pnp/$(PROJECT).board.xml

setup:
	openpnp-tools config apply
	openpnp-tools assign --reset-unused pnp/feeders.csv

save:
	openpnp-tools config save -m "tuning: update from $(PROJECT)"

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

## Example

[Granit](https://laenzlinger.github.io/granit/) is a hardware project using
openpnp-tools for its pick-and-place workflow. See its
[hardware/Makefile](https://github.com/laenzlinger/granit/blob/main/hardware/Makefile)
and [pnp/](https://github.com/laenzlinger/granit/tree/main/hardware/pnp) directory
for a real-world setup. The
[interactive feeder map](https://laenzlinger.github.io/granit/latest/feeder-map.html)
is also deployed to the site.

## Acknowledgments

The interactive feeder map was inspired by
[psypnp](https://inductive-kickback.com/2020/10/psypnp-for-openpnp/) by
Pat Deegan — a collection of OpenPnP scripting utilities for feeder management
and job setup.

## License

GPL-3.0-or-later — aligned with [OpenPnP](https://github.com/openpnp/openpnp).
