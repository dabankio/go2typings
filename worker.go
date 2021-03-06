package go2types

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"text/template"
)

// const
const (
	DefaultWorkerTemplate = `{{.FileDoc}}

{{if .Namespace}}export namespace {{.Namespace}} { {{end}}
{{range .Structs}}{{.MustRender}}{{end}}
{{if .Namespace}}}{{end}}
`
)

// NewWorker .
func NewWorker() *Worker {
	return &Worker{
		seen: map[reflect.Type]*Struct{},
	}
}

// WorkerRenderContext .
type WorkerRenderContext struct {
	FileDoc   string
	Namespace string
	Ident     string
	Structs   []*Struct
}

// Worker .
type Worker struct {
	Namespace string

	structs []*Struct
	seen    map[reflect.Type]*Struct
}

// Add .
func (s *Worker) Add(v ...interface{}) {
	for _, x := range v {
		s.AddWithName(x, "")
	}
}

// AddWithName .
func (s *Worker) AddWithName(v interface{}, name string) *Struct {
	var t reflect.Type
	switch v := v.(type) {
	case reflect.Type:
		t = v
	case reflect.Value:
		t = v.Type()
	default:
		t = reflect.TypeOf(v)
	}

	return s.addType(t, name, "")
}

func (s *Worker) addType(t reflect.Type, name, namespace string) (out *Struct) {
	t = indirect(t)

	if out = s.seen[t]; out != nil {
		return out
	}
	out = MakeStruct(t, name, namespace)
	fullName := out.Name
	out.Type = RegularType

	var numField int
	if t.Kind() == reflect.Struct {
		numField = t.NumField()
		out.Fields = make([]*Field, 0, numField)
	}
	s.seen[t] = out
	for i := 0; i < numField; i++ {
		sf := t.Field(i)

		parsedField := ParseField(sf, CustomTypeMap, out.T)
		if parsedField.Omitted {
			continue
		}

		if !parsedField.Omitted && !parsedField.Anomynous {
			fullFieldName := sf.Type.Name()
			if fullFieldName == "" {
				fullFieldName = sf.Name + fullName
			}
			s.visitType(sf.Type, fullFieldName, namespace)
		}

		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {

			// extendsType := s.seen[sf.Type].Name
			out.InheritedType = append(out.InheritedType, sf.Type.Name())
			continue
		}
		out.Fields = append(out.Fields, parsedField)
	}

	s.structs = append(s.structs, out)
	return
}

// TypescriptEnumMember .
type TypescriptEnumMember struct {
	Name    string
	Value   string
	Comment string
}

func (s *Worker) visitType(t reflect.Type, name, namespace string) {
	k := t.Kind()
	switch {
	case k == reflect.Ptr:
		t = indirect(t)
		s.visitType(t, name, namespace)
	case k == reflect.Struct:
		if isDate(t) {
			break
		}
		if t.Name() != "" {
			name = t.Name()
		}
		s.addType(t, name, namespace)
	case k == reflect.Slice || k == reflect.Array:
		s.visitType(t.Elem(), name, namespace)
	case k == reflect.Map:
		s.visitType(t.Elem(), name, namespace)
		s.visitType(t.Key(), name, namespace)
	case (isNumber(k) || k == reflect.String) && isEnum(t):
		s.AddTypeEnum(t, "", "")
	}
}

// AddTypeEnum add enum
func (s *Worker) AddTypeEnum(t reflect.Type, name, namespace string, pkgNames ...string) (out *Struct) {
	t = indirect(t)
	if out = s.seen[t]; out != nil {
		return out
	}
	out = MakeStruct(t, name, namespace)
	out.Values = getEnumStringValues(t, pkgNames...)
	out.Doc = getDoc(t.PkgPath(), t.Name(), docTypeType)
	out.Type = Enum
	s.seen[t] = out
	s.structs = append(s.structs, out)
	return
}

// RenderTo .
func (s *Worker) RenderTo(w io.Writer) error {
	ctx := WorkerRenderContext{
		FileDoc:   DefaultFileDoc,
		Namespace: s.Namespace,
		Ident:     "  ",
		Structs:   s.structs,
	}

	classBlockIndent := DefaultIndent
	if s.Namespace == "" {
		classBlockIndent = ""
	}
	fieldIndent := classBlockIndent + DefaultIndent
	for i := 0; i < len(ctx.Structs); i++ {
		ctx.Structs[i].Indent = classBlockIndent
		for j := 0; j < len(ctx.Structs[i].Fields); j++ {
			ctx.Structs[i].Fields[j].Indent = fieldIndent
		}
	}
	return template.Must(template.New("worker_tpl").Parse(DefaultWorkerTemplate)).Execute(w, ctx)
}

// MustGenerateFile .
func (s *Worker) MustGenerateFile(path string) {
	interfacesPath, err := filepath.Abs(path)
	panicIf(err)
	interfacesFile, err := os.Create(interfacesPath)
	panicIf(err)
	err = s.RenderTo(interfacesFile)
	panicIf(err)
	f, err := os.Open(interfacesPath)
	panicIf(err)
	err = f.Close()
	panicIf(err)
}
