package gen

import (
	"github.com/r27153733/fastgozero/tools/fastgoctl/model/sql/template"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util/pathx"
)

func genTag(table Table, in string) (string, error) {
	if in == "" {
		return in, nil
	}

	text, err := pathx.LoadTemplate(category, tagTemplateFile, template.Tag)
	if err != nil {
		return "", err
	}

	output, err := util.With("tag").Parse(text).Execute(map[string]any{
		"field": in,
		"data":  table,
	})
	if err != nil {
		return "", err
	}

	return output.String(), nil
}
