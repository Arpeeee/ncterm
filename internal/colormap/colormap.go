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

// landColor is rendered for fill/masked cells so land areas appear as a neutral tone.
const landColor = "#5c5c4a"

// Render maps a 2D float64 grid to rows of colored terminal characters.
// It uses Unicode half-block characters (▄) to pack two data rows into one
// terminal row, which compensates for the ~2:1 height-to-width aspect ratio
// of terminal cells and keeps geographic proportions correct.
// maxCols subsamples columns; maxRows subsamples rows (0 means no limit).
func Render(data [][]float64, cm Colormap, fillValue float64, maxCols, maxRows int) []string {
	min, max := dataRange(data, fillValue)
	span := max - min
	if span == 0 {
		span = 1
	}

	// Subsample to 2*maxRows data rows so each pair becomes one terminal row.
	dataRows := subsampleDataRows(data, maxRows*2)

	nOut := (len(dataRows) + 1) / 2
	out := make([]string, nOut)
	for i := range out {
		top := dataRows[i*2]
		bot := top
		if i*2+1 < len(dataRows) {
			bot = dataRows[i*2+1]
		}
		out[i] = renderHalfBlockRow(top, bot, min, span, fillValue, cm, maxCols)
	}
	return out
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

// renderHalfBlockRow renders two data rows as one terminal row using ▄ (U+2584).
// The upper pixel maps to background color, the lower pixel to foreground color.
func renderHalfBlockRow(top, bot []float64, min, span, fillValue float64, cm Colormap, maxCols int) string {
	indices := columnIndices(len(top), maxCols)
	var b strings.Builder
	for _, j := range indices {
		topC := cellColor(top[j], min, span, fillValue, cm)
		botJ := j
		if botJ >= len(bot) {
			botJ = len(bot) - 1
		}
		botC := cellColor(bot[botJ], min, span, fillValue, cm)
		b.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color(topC)).
			Foreground(lipgloss.Color(botC)).
			Render("▄"))
	}
	return b.String()
}

func cellColor(v, min, span, fillValue float64, cm Colormap) string {
	if isMasked(v, fillValue) {
		return landColor
	}
	t := (v - min) / span
	n := len(cm.Colors)
	i := int(t * float64(n-1))
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	return cm.Colors[i]
}

func subsampleDataRows(data [][]float64, max int) [][]float64 {
	if max <= 0 || len(data) <= max {
		return data
	}
	out := make([][]float64, max)
	for i := range out {
		out[i] = data[int(float64(i)*float64(len(data))/float64(max))]
	}
	return out
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
