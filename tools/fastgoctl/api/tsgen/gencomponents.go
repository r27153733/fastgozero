package tsgen

import (
	_ "embed"
	"path"
	"strings"
	"text/template"

	"github.com/r27153733/fastgozero/tools/fastgoctl/api/spec"
	apiutil "github.com/r27153733/fastgozero/tools/fastgoctl/api/util"
	"github.com/r27153733/fastgozero/tools/fastgoctl/internal/version"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util/pathx"
)

//go:embed components.tpl
var componentsTemplate string

// BuildTypes generates the typescript code for the types.
func BuildTypes(types []spec.Type) (string, error) {
	var builder strings.Builder
	first := true
	for _, tp := range types {
		if first {
			first = false
		} else {
			builder.WriteString("\n")
		}
		if err := writeType(&builder, tp); err != nil {
			return "", apiutil.WrapErr(err, "Type "+tp.Name()+" generate error")
		}
	}

	return builder.String(), nil
}

func genComponents(dir string, api *spec.ApiSpec) error {
	types := api.Types
	if len(types) == 0 {
		return nil
	}

	val, err := BuildTypes(types)
	if err != nil {
		return err
	}

	outputFile := apiutil.ComponentName(api) + ".ts"
	filename := path.Join(dir, outputFile)
	if err := pathx.RemoveIfExist(filename); err != nil {
		return err
	}

	fp, created, err := apiutil.MaybeCreateFile(dir, ".", outputFile)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	defer fp.Close()

	t := template.Must(template.New("componentsTemplate").Parse(componentsTemplate))
	return t.Execute(fp, map[string]string{
		"componentTypes": val,
		"version":        version.BuildVersion,
	})
}
