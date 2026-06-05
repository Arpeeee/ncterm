package tui

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Arpeeee/ncterm/internal/cf"
	"github.com/Arpeeee/ncterm/internal/colormap"
	"github.com/Arpeeee/ncterm/internal/nc"
)

// Viewer is the bubbletea model for ncterm view.
type Viewer struct {
	file     *nc.File
	vars     []nc.Variable // variables with ≥2 dimensions
	selected int

	// Identified axis dimension names for the selected variable
	latDim, lonDim, timeDim, levelDim string
	timeValues                         []float64
	levelSize                          int

	// Navigation state
	timeIdx  int
	levelIdx int
	cmIdx    int

	// Current slice
	slice     [][]float64
	sliceMin  float64
	sliceMax  float64
	sliceMean float64
	fillValue float64

	showInspect   bool
	width, height int
}

// NewViewer creates a Viewer for the given file.
func NewViewer(f *nc.File) *Viewer {
	v := &Viewer{file: f, fillValue: 9.96921e+36}
	v.vars = multiDimVars(f)
	if len(v.vars) > 0 {
		v.selectVar(0)
	}
	return v
}

func (m *Viewer) Init() tea.Cmd { return nil }

func (m *Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up":
			m.selectVar((m.selected - 1 + len(m.vars)) % len(m.vars))
		case "down":
			m.selectVar((m.selected + 1) % len(m.vars))
		case "left":
			if m.timeIdx > 0 {
				m.timeIdx--
				m.reloadSlice()
			}
		case "right":
			if m.timeIdx < m.timeLen()-1 {
				m.timeIdx++
				m.reloadSlice()
			}
		case "[":
			if m.levelIdx > 0 {
				m.levelIdx--
				m.reloadSlice()
			}
		case "]":
			if m.levelIdx < m.levelSize-1 {
				m.levelIdx++
				m.reloadSlice()
			}
		case "c":
			m.cmIdx = (m.cmIdx + 1) % len(colormap.All)
		case "i":
			m.showInspect = !m.showInspect
		}
	}
	return m, nil
}

func (m *Viewer) View() string {
	if len(m.vars) == 0 {
		return "No displayable variables found.\n\nPress q to quit."
	}

	leftW := 22
	rightW := m.width - leftW - 3
	if rightW < 20 {
		rightW = 20
	}

	left := m.renderVarList(leftW)
	var right string
	if m.showInspect {
		right = renderVariable(m.vars[m.selected])
	} else {
		right = m.renderSlice(rightW, m.height-2)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftW).Render(left),
		sepStyle.Render(" │ "),
		lipgloss.NewStyle().Width(rightW).Render(right),
	)

	return body + "\n" + m.renderStatus() + "\n" + renderKeyHints(m)
}

func (m *Viewer) renderVarList(width int) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Variables") + "\n\n")
	for i, v := range m.vars {
		name := truncate(v.Name, width-3)
		if i == m.selected {
			b.WriteString(lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color("12")).Render("> "+name) + "\n")
		} else {
			b.WriteString("  " + name + "\n")
		}
	}
	return b.String()
}

func (m *Viewer) renderSlice(width, height int) string {
	if m.slice == nil {
		return "No spatial data\n(select a variable with lat/lon dims)"
	}

	// Reserve 3 lines for the frame: top border, bottom border, lon label row.
	const frameLines = 3
	mapHeight := height - frameLines
	if mapHeight < 1 {
		mapHeight = 1
	}

	// Maintain geographic aspect ratio.
	// Half-block chars give 2 data pixels per terminal row, and terminal chars are
	// ~2:1 (h:w), so the visual pixel ratio is cols / (2 * rows).
	// Target: cols / (2 * rows) = lonN / latN
	latN := float64(len(m.slice))
	lonN := float64(0)
	if latN > 0 {
		lonN = float64(len(m.slice[0]))
	}
	cols, rows := width, mapHeight
	if latN > 0 && lonN > 0 {
		aspect := lonN / latN
		if float64(cols) > aspect*2.0*float64(rows) {
			cols = int(math.Round(aspect * 2.0 * float64(rows)))
		} else {
			rows = int(math.Round(float64(cols) / (aspect * 2.0)))
		}
		if cols < 1 {
			cols = 1
		}
		if rows < 1 {
			rows = 1
		}
	}

	mapRows := colormap.Render(m.slice, colormap.All[m.cmIdx], m.fillValue, cols, rows)

	latMax, latMin := m.latExtent()
	lonMin, lonMax := m.lonExtent()

	topLat := formatLat(latMax)
	botLat := formatLat(latMin)
	labelW := len(topLat)
	if len(botLat) > labelW {
		labelW = len(botLat)
	}
	topLat = fmt.Sprintf("%-*s", labelW, topLat)
	botLat = fmt.Sprintf("%-*s", labelW, botLat)
	indent := strings.Repeat(" ", labelW)
	border := strings.Repeat("─", cols)

	var b strings.Builder
	b.WriteString(topLat + " ┌" + border + "┐\n")
	for _, row := range mapRows {
		b.WriteString(indent + " │" + row + "│\n")
	}
	b.WriteString(botLat + " └" + border + "┘\n")

	// Lon labels: left-aligned at start of border, right-aligned at end.
	leftLon := formatLon(lonMin)
	rightLon := formatLon(lonMax)
	innerW := cols + 2 // interior width including │ chars
	pad := innerW - len(leftLon) - len(rightLon)
	if pad < 1 {
		pad = 1
	}
	b.WriteString(indent + " " + leftLon + strings.Repeat(" ", pad) + rightLon)

	return b.String()
}

func (m *Viewer) latExtent() (latMax, latMin float64) {
	if m.latDim != "" {
		if v := findVar(m.file, m.latDim); v != nil {
			vals := v.Float64s()
			if len(vals) > 0 {
				a, b := vals[0], vals[len(vals)-1]
				if a > b {
					return a, b
				}
				return b, a
			}
		}
	}
	return 90, -90
}

func (m *Viewer) lonExtent() (lonMin, lonMax float64) {
	if m.lonDim != "" {
		if v := findVar(m.file, m.lonDim); v != nil {
			vals := v.Float64s()
			if len(vals) > 0 {
				return vals[0], vals[len(vals)-1]
			}
		}
	}
	return -180, 180
}

func formatLat(deg float64) string {
	if deg >= 0 {
		return fmt.Sprintf("%.1f°N", deg)
	}
	return fmt.Sprintf("%.1f°S", -deg)
}

func formatLon(deg float64) string {
	if deg >= 0 {
		return fmt.Sprintf("%.0f°E", deg)
	}
	return fmt.Sprintf("%.0f°W", -deg)
}

func (m *Viewer) renderStatus() string {
	timeLabel := fmt.Sprintf("t=%d", m.timeIdx)
	if m.timeDim != "" && m.timeIdx < len(m.timeValues) {
		if tv := findVar(m.file, m.timeDim); tv != nil {
			if units, ok := tv.Attrs["units"].(string); ok {
				if base, step, err := cf.ParseTimeUnits(units); err == nil {
					timeLabel = cf.FormatTime(base, step, m.timeValues[m.timeIdx])
				}
			}
		}
	}
	return fmt.Sprintf("  min=%.4g  max=%.4g  mean=%.4g  %s  lev=%d  cm=%s",
		m.sliceMin, m.sliceMax, m.sliceMean,
		timeLabel, m.levelIdx, colormap.All[m.cmIdx].Name)
}

func (m *Viewer) selectVar(idx int) {
	m.selected = idx
	m.timeIdx, m.levelIdx = 0, 0
	m.latDim, m.lonDim, m.timeDim, m.levelDim = "", "", "", ""
	m.timeValues = nil
	m.levelSize = 1

	v := m.vars[idx]
	for _, dimName := range v.Dims {
		coord := findVar(m.file, dimName)
		if coord == nil {
			continue
		}
		switch cf.DetectAxis(*coord) {
		case cf.AxisLat:
			m.latDim = dimName
		case cf.AxisLon:
			m.lonDim = dimName
		case cf.AxisTime:
			m.timeDim = dimName
			m.timeValues = coord.Float64s()
		case cf.AxisLevel:
			m.levelDim = dimName
			m.levelSize = m.file.Dims[dimName]
		}
	}

	if m.latDim == "" || m.lonDim == "" {
		m.latDim, m.lonDim = guessSpatialDims(v.Dims)
	}
	if m.timeDim == "" {
		m.timeDim = guessDim(v.Dims, []string{"time", "t"})
	}

	m.fillValue = getFillValue(v)
	m.reloadSlice()
}

func (m *Viewer) reloadSlice() {
	if len(m.vars) == 0 {
		return
	}
	v := m.vars[m.selected]
	outer := map[string]int{}
	if m.timeDim != "" {
		outer[m.timeDim] = m.timeIdx
	}
	if m.levelDim != "" {
		outer[m.levelDim] = m.levelIdx
	}
	m.slice = v.Slice2D(m.latDim, m.lonDim, outer)
	if m.slice != nil {
		// Ensure north is at top (row 0). Many files store lat ascending
		// (south-first); flip rows so the display always matches convention.
		if latVar := findVar(m.file, m.latDim); latVar != nil {
			vals := latVar.Float64s()
			if len(vals) > 1 && vals[0] < vals[len(vals)-1] {
				for i, j := 0, len(m.slice)-1; i < j; i, j = i+1, j-1 {
					m.slice[i], m.slice[j] = m.slice[j], m.slice[i]
				}
			}
		}
		m.sliceMin, m.sliceMax, m.sliceMean = colormap.Stats(m.slice, m.fillValue)
	}
}

func (m *Viewer) timeLen() int {
	if m.timeDim == "" {
		return 1
	}
	return m.file.Dims[m.timeDim]
}

func renderKeyHints(m *Viewer) string {
	inspect := "i inspect"
	if m.showInspect {
		inspect = "i back"
	}
	parts := []string{"↑↓ var", "←→ time", "[] level", "c colormap", inspect, "q quit"}
	return sepStyle.Render("  " + strings.Join(parts, "  │  "))
}

func multiDimVars(f *nc.File) []nc.Variable {
	var out []nc.Variable
	for _, v := range f.Variables {
		if len(v.Dims) >= 2 {
			out = append(out, v)
		}
	}
	return out
}

func findVar(f *nc.File, name string) *nc.Variable {
	for i := range f.Variables {
		if f.Variables[i].Name == name {
			return &f.Variables[i]
		}
	}
	return nil
}

func guessSpatialDims(dims []string) (lat, lon string) {
	latNames := map[string]bool{"lat": true, "latitude": true, "y": true, "rlat": true}
	lonNames := map[string]bool{"lon": true, "longitude": true, "x": true, "rlon": true}
	for _, d := range dims {
		dl := strings.ToLower(d)
		if latNames[dl] {
			lat = d
		}
		if lonNames[dl] {
			lon = d
		}
	}
	return
}

func guessDim(dims, candidates []string) string {
	for _, d := range dims {
		dl := strings.ToLower(d)
		for _, c := range candidates {
			if dl == c {
				return d
			}
		}
	}
	return ""
}

func getFillValue(v nc.Variable) float64 {
	for _, key := range []string{"_FillValue", "missing_value"} {
		if val, ok := v.Attrs[key]; ok {
			switch fv := val.(type) {
			case float32:
				return float64(fv)
			case float64:
				return fv
			case []float32:
				if len(fv) > 0 {
					return float64(fv[0])
				}
			case []float64:
				if len(fv) > 0 {
					return fv[0]
				}
			}
		}
	}
	return 9.96921e+36
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
