package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Arpeeee/ncterm/internal/cf"
	"github.com/Arpeeee/ncterm/internal/nc"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	keyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	sepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// RenderInfo returns a formatted summary of the NetCDF file for ncterm info output.
func RenderInfo(f *nc.File) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("File: "+f.Path) + "\n\n")

	if len(f.GlobalAttrs) > 0 {
		b.WriteString(headerStyle.Render("Global Attributes") + "\n")
		for k, v := range f.GlobalAttrs {
			b.WriteString(fmt.Sprintf("  %s = %v\n", keyStyle.Render(k), v))
		}
		b.WriteString("\n")
	}

	b.WriteString(headerStyle.Render("Dimensions") + "\n")
	for name, size := range f.Dims {
		b.WriteString(fmt.Sprintf("  %s = %d\n", dimStyle.Render(name), size))
	}
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Variables") + "\n")
	b.WriteString(sepStyle.Render(strings.Repeat("─", 60)) + "\n")
	for _, v := range f.Variables {
		b.WriteString(renderVariable(v))
	}

	return b.String()
}

// renderVariable formats one variable's metadata block.
// Shared by RenderInfo and the viewer's inspect overlay.
func renderVariable(v nc.Variable) string {
	var b strings.Builder

	shape := formatShape(v.Dims, v.Shape)
	b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
		keyStyle.Render(v.Name), v.Type, dimStyle.Render(shape)))

	axisType := cf.DetectAxis(v)
	if axisType != cf.AxisUnknown {
		b.WriteString(fmt.Sprintf("    axis: %s\n", axisType))
	}
	if name, ok := v.Attrs["long_name"].(string); ok {
		b.WriteString(fmt.Sprintf("    long_name: %s\n", name))
	}
	if name, ok := v.Attrs["standard_name"].(string); ok {
		b.WriteString(fmt.Sprintf("    standard_name: %s\n", name))
	}
	if units, ok := v.Attrs["units"].(string); ok {
		b.WriteString(fmt.Sprintf("    units: %s\n", units))
	}
	for _, key := range []string{"_FillValue", "missing_value"} {
		if val, ok := v.Attrs[key]; ok {
			b.WriteString(fmt.Sprintf("    %s: %v\n", key, val))
		}
	}
	if axisType == cf.AxisTime {
		if units, ok := v.Attrs["units"].(string); ok {
			if base, step, err := cf.ParseTimeUnits(units); err == nil {
				b.WriteString(fmt.Sprintf("    time base: %s, step: %s\n",
					base.Format("2006-01-02"), step))
			}
		}
	}

	return b.String()
}

func formatShape(dims []string, shape []int) string {
	parts := make([]string, len(dims))
	for i, d := range dims {
		parts[i] = fmt.Sprintf("%s=%d", d, shape[i])
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
