package nc

import (
	"fmt"
	"reflect"

	"github.com/batchatco/go-native-netcdf/netcdf"
	"github.com/batchatco/go-native-netcdf/netcdf/api"
)

// Variable holds metadata and raw data for one NetCDF variable.
type Variable struct {
	Name   string
	Type   string
	Dims   []string
	Shape  []int
	Attrs  map[string]interface{}
	values interface{} // nested Go slices from library, e.g. [][][]float32
}

// File is an open NetCDF file with pre-loaded metadata.
type File struct {
	Path        string
	Dims        map[string]int
	Variables   []Variable
	GlobalAttrs map[string]interface{}
	handler     api.Group
}

// Open reads metadata from a NetCDF file. Call Close when done.
func Open(path string) (*File, error) {
	h, err := netcdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	f := &File{Path: path, handler: h}
	if err := f.load(); err != nil {
		h.Close()
		return nil, err
	}
	return f, nil
}

func (f *File) Close() {
	f.handler.Close()
}

func (f *File) load() error {
	f.GlobalAttrs = copyAttrs(f.handler.Attributes())

	f.Dims = make(map[string]int)
	for _, name := range f.handler.ListDimensions() {
		if n, ok := f.handler.GetDimension(name); ok {
			f.Dims[name] = int(n)
		}
	}

	for _, name := range f.handler.ListVariables() {
		v, err := f.handler.GetVariable(name)
		if err != nil {
			return fmt.Errorf("load variable %s: %w", name, err)
		}

		shape := make([]int, len(v.Dimensions))
		for i, d := range v.Dimensions {
			shape[i] = f.Dims[d]
		}

		f.Variables = append(f.Variables, Variable{
			Name:   name,
			Type:   reflect.TypeOf(v.Values).String(),
			Dims:   v.Dimensions,
			Shape:  shape,
			Attrs:  copyAttrs(v.Attributes),
			values: v.Values,
		})
	}
	return nil
}

// Float64s returns all variable values as a flat []float64.
func (v Variable) Float64s() []float64 {
	rv := reflect.ValueOf(v.values)
	out := make([]float64, 0, flatLen(rv))
	collectFloat64s(rv, &out)
	return out
}

// Slice2D extracts a lat×lon 2D slice at the given outer dimension indices.
// outerIndices maps dimension name → index for all dims that precede latDim.
// Assumes latDim comes before lonDim in v.Dims (standard CF convention order).
func (v Variable) Slice2D(latDim, lonDim string, outerIndices map[string]int) [][]float64 {
	if !hasDim(v.Dims, latDim) || !hasDim(v.Dims, lonDim) {
		return nil
	}

	rv := reflect.ValueOf(v.values)
	for _, dim := range v.Dims {
		if dim == latDim {
			break
		}
		rv = rv.Index(outerIndices[dim]) // defaults to 0 if key absent
	}

	rows := rv.Len()
	if rows == 0 {
		return nil
	}

	result := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		row := rv.Index(i)
		cols := row.Len()
		result[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			result[i][j] = toFloat64(row.Index(j))
		}
	}
	return result
}

func hasDim(dims []string, name string) bool {
	for _, d := range dims {
		if d == name {
			return true
		}
	}
	return false
}

func flatLen(rv reflect.Value) int {
	if rv.Kind() != reflect.Slice || rv.Len() == 0 {
		return 1
	}
	return rv.Len() * flatLen(rv.Index(0))
}

func collectFloat64s(rv reflect.Value, out *[]float64) {
	if rv.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			collectFloat64s(rv.Index(i), out)
		}
		return
	}
	*out = append(*out, toFloat64(rv))
}

func toFloat64(rv reflect.Value) float64 {
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	default:
		return 0
	}
}

func copyAttrs(m api.AttributeMap) map[string]interface{} {
	if m == nil {
		return make(map[string]interface{})
	}
	keys := m.Keys()
	out := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		if v, ok := m.Get(k); ok {
			out[k] = v
		}
	}
	return out
}
