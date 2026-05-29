# ncterm

Terminal-based NetCDF viewer. No X11, no GUI, runs over SSH.

## Problem

`ncview` requires X11 forwarding — unusable on remote HPC clusters with slow networks or no display server.  
`ncdump` dumps raw text — unreadable for large files.

`ncterm` fills this gap with two modes that run in any standard terminal:

```
ncterm info <file.nc>   # structured metadata summary
ncterm view <file.nc>   # interactive TUI with keyboard navigation
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [batchatco/go-native-netcdf](https://github.com/batchatco/go-native-netcdf) | Pure-Go NetCDF-3/4 reader. No `libnetcdf` required — single binary deploy. |
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | Elm-architecture TUI framework. Handles keyboard events and render loop. |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling — colors, layout, borders. |

No CGo, no system libraries. Copy the binary to any Linux machine and it works.

## Install

```bash
# From source
git clone https://github.com/user/ncterm
cd ncterm
go build -o ncterm ./cmd/ncterm

# Or go install
go install github.com/user/ncterm@latest
```

Requires Go 1.22+.

## Usage

### `ncterm info` — inspect a file

```bash
ncterm info data.nc
```

Prints a structured summary:

```
File: data.nc

Global Attributes
  Conventions = CF-1.6
  source = ERA5 reanalysis

Dimensions
  time = 8760
  lat  = 721
  lon  = 1440
  lev  = 37

Variables
────────────────────────────────────────────────────────────
  t2m  []float32  (time=8760, lat=721, lon=1440)
    long_name: 2 metre temperature
    units: K
    _FillValue: 9.96921e+36

  z    []float32  (time=8760, lev=37, lat=721, lon=1440)
    long_name: Geopotential
    units: m**2 s**-2
```

Replaces `ncdump -h` with formatted, readable output. Parses CF convention
attributes (`units`, `standard_name`, `axis`, `_FillValue`) automatically.

### `ncterm view` — interactive viewer

```bash
ncterm view data.nc
```

Opens a full-screen TUI:

```
Variables          │ ░░▒▒▓▓██████▓▓▒▒░░░░▒▒▓▓████
                   │ ░░░▒▒▓▓█████████▓▓▒▒░░░░▒▒▓▓
> t2m              │ ▒▒▓▓████████████████▓▓▒▒░░░░
  z                │ ░▒▒▓▓███████████▓▓▒▒░░░░▒▒▓▓
  u10              │ ▒▒░░▒▒▓▓████████████▓▓▒▒░░░░
  v10              │
                   │
  min=-12.3  max=38.7  mean=14.2  2023-06-15 12:00  lev=0  cm=viridis
  ↑↓ var  │  ←→ time  │  [] level  │  c colormap  │  i inspect  │  q quit
```

#### Keyboard controls

| Key | Action |
|-----|--------|
| `↑` `↓` | Navigate variable list |
| `←` `→` | Step through time axis |
| `[` `]` | Step through level / depth axis |
| `c` | Cycle colormap (viridis → plasma → gray) |
| `i` | Toggle inspect overlay (variable attributes) |
| `q` | Quit |

## CF Convention Support

Automatically detects and uses:

- `axis` attribute (`T`, `X`, `Y`, `Z`) — identifies time, lat, lon, level dimensions
- `standard_name` — fallback axis detection
- `units` on time dimension — displayed as human-readable dates (`2023-06-15 12:00`)
- `_FillValue` / `missing_value` — masked in colormap rendering and statistics

## Supported Formats

- NetCDF-4 / HDF5
- NetCDF-3 / CDF

## Project Layout

```
cmd/ncterm/        entry point, subcommand routing
internal/nc/       NetCDF read layer (wraps go-native-netcdf)
internal/cf/       CF convention parsing (time, axis detection)
internal/colormap/ float64 grid → colored terminal output
internal/tui/      bubbletea models (info display, slice viewer)
docs/guide.md      Go language intro and environment setup
```

## License

MIT
