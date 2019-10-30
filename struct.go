package go2typings

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

type Struct struct {
	ReferenceName string
	Namespace     string
	Name          string
	Fields        []*Field
	InheritedType []string
	T             reflect.Type
}

const template = `  //%s.%s
  export interface %s %s{%s}
`

func (s *Struct) RenderTo(opts *Options, w io.Writer) (err error) {
	extendsType := ""
	if len(s.InheritedType) != 0 {
		extendsType = fmt.Sprintf("extends %s ", strings.Join(s.InheritedType, ", "))
	}

	fields := ""
	for n, field := range s.Fields {
		name, t := Type(field)
		fields += fmt.Sprintf("\n    %s: %s;", name, t)
		if n == len(s.Fields)-1 {
			fields += "\n  "
		}
	}
	_, err = fmt.Fprintf(w, template, s.T.PkgPath(), s.T.Name(), s.Name, extendsType, fields)
	return
}