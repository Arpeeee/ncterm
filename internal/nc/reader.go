package nc

import (
	"fmt"
	"reflect"

	"github.com/batchatco/go-native-netcdf/netcdf"
	"github.com/batchatco/go-native-netcdf/netcdf/api"
)

// Variable holds metadata and lazy data access for one NetCDF variable.
type Variable struct {
	Name   string
	Type   string
	Dims   []string
	Shape  []int
	Attrs  map[string]interface{}
	values interface{}   // fully loaded flat values (coord vars only)
	getter api.VarGetter // lazy accessor; non-nil for all variables
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
// Variable data is NOT loaded at open time; it is fetched on first access.
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

	// Coordinate variables share their name with a dimension (lat, lon, time, level).
	// They are always 1D and tiny, so we load their values eagerly.
	coordNames := make(map[string]bool, len(f.Dims))
	for name := range f.Dims {
		coordNames[name] = true
	}

	for _, name := range f.handler.ListVariables() {
		getter, err := f.handler.GetVarGetter(name)
		if err != nil {
			return fmt.Errorf("get variable getter %s: %w", name, err)
		}

		dims := getter.Dimensions()
		shape := make([]int, len(dims))
		for i, d := range dims {
			shape[i] = f.Dims[d]
		}
		attrs := copyAttrs(getter.Attributes())

		typeStr := ""
		var values interface{}

		if coordNames[name] && len(dims) == 1 {
			// Coordinate variable: load all values now — they are tiny.
			all, err := getter.GetSlice(0, getter.Len())
			if err == nil {
				values = all
				typeStr = reflect.TypeOf(all).Elem().String()
			}
		} else {
			// Data variable: read one element just for the type label.
			if getter.Len() > 0 {
				if sample, err := getter.GetSlice(0, 1); err == nil {
					typeStr = reflect.TypeOf(sample).Elem().String()
				}
			}
		}

		f.Variables = append(f.Variables, Variable{
			Name:   name,
			Type:   typeStr,
			Dims:   dims,
			Shape:  shape,
			Attrs:  attrs,
			values: values,
			getter: getter,
		})
	}
	return nil
}

// Float64s returns all variable values as a flat []float64.
// For large data variables this reads the entire variable from disk on first call.
func (v Variable) Float64s() []float64 {
	rv := reflect.ValueOf(v.values)
	if !rv.IsValid() {
		if v.getter == nil {
			return nil
		}
		all, err := v.getter.GetSlice(0, v.getter.Len())
		if err != nil {
			return nil
		}
		rv = reflect.ValueOf(all)
	}
	out := make([]float64, 0, flatLen(rv))
	collectFloat64s(rv, &out)
	return out
}

// Slice2D extracts a lat×lon 2D slice at the given outer dimension indices.
// outerIndices maps dimension name → index for all dims that are not lat/lon.
// Uses GetSliceMD to read only the required 2D subset from disk.
func (v Variable) Slice2D(latDim, lonDim string, outerIndices map[string]int) [][]float64 {
	if !hasDim(v.Dims, latDim) || !hasDim(v.Dims, lonDim) {
		return nil
	}

	if v.getter != nil {
		return v.slice2DFromGetter(latDim, lonDim, outerIndices)
	}
	return v.slice2DFromValues(latDim, lonDim, outerIndices)
}

// slice2DFromGetter reads only the [latN × lonN] subset using GetSliceMD.
func (v Variable) slice2DFromGetter(latDim, lonDim string, outerIndices map[string]int) [][]float64 {
	begin := make([]int64, len(v.Dims))
	end := make([]int64, len(v.Dims))
	for i, dim := range v.Dims {
		if dim == latDim || dim == lonDim {
			end[i] = int64(v.Shape[i])
		} else {
			idx := int64(outerIndices[dim]) // 0 if key absent
			begin[i] = idx
			end[i] = idx + 1
		}
	}

	data, err := v.getter.GetSliceMD(begin, end)
	if err != nil {
		return nil
	}

	// Navigate outer dims (each of length 1 after GetSliceMD slicing) to reach [lat][lon].
	rv := reflect.ValueOf(data)
	for _, dim := range v.Dims {
		if dim == latDim {
			break
		}
		rv = rv.Index(0)
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

// slice2DFromValues navigates pre-loaded nested slices via reflection.
func (v Variable) slice2DFromValues(latDim, lonDim string, outerIndices map[string]int) [][]float64 {
	rv := reflect.ValueOf(v.values)
	for _, dim := range v.Dims {
		if dim == latDim {
			break
		}
		rv = rv.Index(outerIndices[dim])
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
