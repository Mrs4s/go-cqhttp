package main

import "html/template"

func (g *generator) genSupported(routers []Router) {
	var v11, v12 []string // for onebot v12 get_supported_actions
	for _, router := range routers {
		if len(router.PathV11) > 0 {
			v11 = append(v11, router.PathV11...)
		}
		if len(router.PathV11) > 0 {
			v12 = append(v12, router.PathV12...)
		}
		if len(router.Path) > 0 {
			v11 = append(v11, router.Path...)
			v12 = append(v12, router.Path...)
		}
	}

	type S struct {
		V11 []string
		V12 []string
	}

	tmpl, err := template.New("").Parse(supportedTemplete)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(g.out, &S{V11: v11, V12: v12})
	if err != nil {
		panic(err)
	}
}

const supportedTemplete = `
var supportedV11 = []string{
	{{range .V11}}	"{{.}}",
{{end}}
}

var supportedV12 = []string{
	{{range .V12}}	"{{.}}",
{{end}}
}`
