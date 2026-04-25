# openpnp-feeder-map

[![CI](https://github.com/laenzlinger/openpnp-feeder-map/actions/workflows/ci.yml/badge.svg)](https://github.com/laenzlinger/openpnp-feeder-map/actions/workflows/ci.yml)

Generates an interactive HTML feeder map for an [OpenPnP](https://openpnp.org/) job.

Given a job file, it reads the referenced board placements and matches them against the feeders configured in `machine.xml`, producing a self-contained HTML page that shows:

- **Visual map** of feeder positions on the machine bed (pan & zoom, click to highlight)
- **Machine bed outline** derived from axis soft limits
- **Strip feeder outlines** showing tape width and feed direction
- **Needed tape length** — highlighted portion of each strip based on placement count × part pitch
- **Job feeders** — which feeders are needed, what part they supply, and how many placements
- **Missing parts** — job parts that have no feeder assigned
- **Unused feeders** — enabled feeders not needed by this job
- **Tape info** — feeder type, tape type, tape width
- **Light/dark mode** — follows system preference, styled to match [PaperMod](https://github.com/adityatelange/hugo-PaperMod)
- **Job focus toggle** — show only job-relevant feeders or the full machine
- **Resizable sidebar** — drag to adjust

Designed for use with strip feeders and push-pull feeders on machines like the [Lumen PnP](https://opulo.io/).

## How it works

```
job.xml ──→ board.xml ──→ list of part-ids + quantities
                                    │
machine.xml ──→ feeders ────────────┤
                                    ▼
                          match parts → feeders
                                    │
                                    ▼
                          feeder-map.html
```

1. The **job file** references one or more board files
2. Each **board file** lists placements with `part-id` and quantity
3. The **machine config** contains feeders with `part-id` and physical locations
4. Parts are matched by exact `part-id` — same as OpenPnP itself

## Usage

```
openpnp-feeder-map [flags] <job.xml>
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-machine` | `~/.openpnp2/machine.xml` | Path to machine.xml |
| `-output` | `feeder-map.html` | Output HTML file path |

### Example

```bash
openpnp-feeder-map -output granit-feeders.html ~/projects/granit/pnp/granit.job.xml
```

Then open `granit-feeders.html` in a browser.

## Supported feeder types

| Type | Visualization |
|------|---------------|
| ReferenceStripFeeder | Tape outline with feed direction, needed length highlight |
| ReferencePushPullFeeder | Location dot with feed direction |
| ReferenceTrayFeeder | Location dot |

## Build

Requires [Go](https://go.dev/) 1.21+. Pin the version with [Mise](https://mise.jdx.dev/):

```bash
make build
```

### Available make targets

```
build          Build the binary
clean          Clean build artifacts
lint           Lint source code
run            Run with example job (set JOB=path/to/job.xml)
test           Run tests
```

## Integration with hugo-kicad-site

The generated HTML is self-contained and can be linked from a
[hugo-kicad-site](https://github.com/laenzlinger/hugo-kicad-site) project page,
or served as a static file alongside the site.

## License

GPL-3.0-or-later — aligned with [OpenPnP](https://github.com/openpnp/openpnp).
