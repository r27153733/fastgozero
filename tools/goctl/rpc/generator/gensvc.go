package generator

import (
	_ "embed"
	"fmt"
	"path/filepath"

	conf "github.com/r27153733/fastgozero/tools/fastgoctl/config"
	"github.com/r27153733/fastgozero/tools/fastgoctl/rpc/parser"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util/format"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util/pathx"
)

//go:embed svc.tpl
var svcTemplate string

// GenSvc generates the servicecontext.go file, which is the resource dependency of a service,
// such as rpc dependency, model dependency, etc.
func (g *Generator) GenSvc(ctx DirContext, _ parser.Proto, cfg *conf.Config) error {
	dir := ctx.GetSvc()
	svcFilename, err := format.FileNamingFormat(cfg.NamingFormat, "service_context")
	if err != nil {
		return err
	}

	fileName := filepath.Join(dir.Filename, svcFilename+".go")
	text, err := pathx.LoadTemplate(category, svcTemplateFile, svcTemplate)
	if err != nil {
		return err
	}

	return util.With("svc").GoFmt(true).Parse(text).SaveTo(map[string]any{
		"imports": fmt.Sprintf(`"%v"`, ctx.GetConfig().Package),
	}, fileName, false)
}
