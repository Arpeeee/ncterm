# ncterm

Terminal-based NetCDF viewer and inspector. No X11, no GUI dependencies, runs over SSH.

## Motivation

`ncview` requires X11 forwarding. On remote HPC clusters with slow network or no display server, it is unusable. `ncdump` outputs raw text dumps that are unreadable for large files. `ncterm` fills this gap with two modes: a fast inspect mode and an interactive TUI viewer, both running in a standard terminal.

## Dependencies

- [go-native-netcdf](https://github.com/batchatco/go-native-netcdf) — pure Go, no libnetcdf required
- [bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling

Single binary, no runtime dependencies. Deployable to HPC by copying the binary.

## Supported Formats

- NetCDF-4 / HDF5 (primary)
- NetCDF-3 / CDF (supported via same library)

## Modes

### Inspect Mode

```
ncterm info <file.nc>
```

Outputs a structured summary of the file:

- Global attributes
- Dimensions and sizes
- Variables: name, type, shape, attributes
- Time axis: parsed from CF convention `units` + `calendar` to human-readable dates
- Missing value / fill value per variable

Designed to replace `ncdump -h` with formatted, readable output.

### Interactive Mode

```
ncterm view <file.nc>
```

TUI interface with keyboard navigation:

- Variable selection list
- Dimension selector (time step, level, other axes)
- 2D slice rendered as colormap using Unicode block characters
- Min / max / mean displayed per slice
- CF-compliant time display when available

## Keyboard Controls (Interactive Mode)

| Key | Action |
|-----|--------|
| `↑↓` | Navigate variable list |
| `←→` | Step through time axis |
| `[` `]` | Step through level / depth axis |
| `c` | Cycle colormap |
| `i` | Toggle inspect overlay |
| `q` | Quit |

## Colormap Rendering

Uses Unicode block characters and braille patterns mapped to a linear colormap. No color terminal required for structure display; color terminals get full 256-color or truecolor palettes.

## CF Convention Support

Parses standard CF attributes:

- `units` + `calendar` on time dimension → display as date string
- `standard_name`, `long_name` → shown in variable list
- `missing_value`, `_FillValue` → masked in statistics and rendering
- `axis` attribute → used to identify lat/lon/level/time dimensions automatically

## MVP Scope

The following is in scope for v0.1:

- `ncterm info` for NetCDF-3 and NetCDF-4
- `ncterm view` with variable selection and time stepping
- 2D slice rendering for variables with lat/lon dimensions
- CF time parsing

The following is deferred:

- Slice export (CSV/JSON)
- Ensemble namelist manager integration
- OPeNDAP / remote file access
- Write operations

## Installation

```bash
go install github.com/<user>/ncterm@latest
```

Or download a pre-built binary from Releases.

## Build from Source

```bash
git clone https://github.com/<user>/ncterm
cd ncterm
go build ./cmd/ncterm
```

## Project Structure

```
ncterm/
├── cmd/
│   └── ncterm/         # entry point, subcommand routing
├── internal/
│   ├── nc/             # NetCDF read layer (wraps go-native-netcdf)
│   ├── cf/             # CF convention parsing (time, axis detection)
│   ├── colormap/       # float → unicode block character mapping
│   └── tui/            # bubbletea models (inspect view, slice view)
└── README.md
```

## Development Principles

- **Simplicity first** — Write the simplest code that works. No speculative abstractions.
- **Single Responsibility** — Each package owns exactly one concern. Functions do one thing.
- **Readability over cleverness** — Clear names, functions under 50 lines, no tricks.
- **MVP discipline** — Build only what the scope requires. Defer everything else.

## License

MIT
