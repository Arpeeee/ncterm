package colormap

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colormap maps normalized float values to terminal colors.
type Colormap struct {
	Name   string
	Colors []string // hex colors ordered low → high value
}

var (
	Viridis = Colormap{
		Name: "viridis",
		Colors: []string{
			"#440154", "#482878", "#3e4989", "#31688e",
			"#26828e", "#1f9e89", "#35b779", "#6ece58",
			"#b5de2b", "#fde725",
		},
	}
	Plasma = Colormap{
		Name: "plasma",
		Colors: []string{
			"#0d0887", "#46039f", "#7201a8", "#9c179e",
			"#bd3786", "#d8576b", "#ed7953", "#fb9f3a",
			"#fdcb26", "#f0f921",
		},
	}
	Grayscale = Colormap{
		Name: "gray",
		Colors: []string{
			"#000000", "#1c1c1c", "#383838", "#555555",
			"#717171", "#8d8d8d", "#aaaaaa", "#c6c6c6",
			"#e2e2e2", "#ffffff",
		},
	}

	All = []Colormap{Viridis, Plasma, Grayscale}
)

// Render maps a 2D float64 grid to rows of colored terminal characters.
// maxCols subsamples columns to fit terminal width; 0 means no limit.
func Render(data [][]float64, cm Colormap, fillValue float64, maxCols int) []string {
	min, max := dataRange(data, fillValue)
	span := max - min
	if span == 0 {
		span = 1
	}

	rows := make([]string, len(data))
	for i, row := range data {
		rows[i] = renderRow(row, min, span, fillValue, cm, maxCols)
	}
	return rows
}

// Stats returns min, max, and mean of data, ignoring fill values and NaN.
func Stats(data [][]float64, fillValue float64) (min, max, mean float64) {
	min = math.Inf(1)
	max = math.Inf(-1)
	sum, count := 0.0, 0

	for _, row := range data {
		for _, v := range row {
			if isMasked(v, fillValue) {
				continue
			}
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
			sum += v
			count++
		}
	}

	if count == 0 {
		return 0, 0, 0
	}
	return min, max, sum / float64(count)
}

func renderRow(row []float64, min, span, fillValue float64, cm Colormap, maxCols int) string {
	indices := columnIndices(len(row), maxCols)
	var b strings.Builder
	for _, j := range indices {
		v := row[j]
		if isMasked(v, fillValue) {
			b.WriteString(" ")
			continue
		}
		t := (v - min) / span
		b.WriteString(colorCell(t, cm))
	}
	return b.String()
}

func columnIndices(total, maxCols int) []int {
	if maxCols <= 0 || total <= maxCols {
		idx := make([]int, total)
		for i := range idx {
			idx[i] = i
		}
		return idx
	}
	idx := make([]int, maxCols)
	for i := range idx {
		idx[i] = int(float64(i) * float64(total) / float64(maxCols))
	}
	return idx
}

func colorCell(t float64, cm Colormap) string {
	n := len(cm.Colors)
	i := int(t * float64(n-1))
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	return lipgloss.NewStyle().Background(lipgloss.Color(cm.Colors[i])).Render(" ")
}

func dataRange(data [][]float64, fillValue float64) (min, max float64) {
	min = math.Inf(1)
	max = math.Inf(-1)
	for _, row := range data {
		for _, v := range row {
			if isMasked(v, fillValue) {
				continue
			}
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}
	if math.IsInf(min, 1) {
		return 0, 1
	}
	return
}

func isMasked(v, fillValue float64) bool {
	return math.IsNaN(v) || math.Abs(v-fillValue) < math.Abs(fillValue)*1e-5
}
