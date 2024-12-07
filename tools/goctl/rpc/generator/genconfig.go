package generator

import (
	_ "embed"
	"os"
	"path/filepath"

	conf "github.com/r27153733/fastgozero/tools/fastgoctl/config"
	"github.com/r27153733/fastgozero/tools/fastgoctl/rpc/parser"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util/format"
	"github.com/r27153733/fastgozero/tools/fastgoctl/util/pathx"
)

//go:embed config.tpl
var configTemplate string

// GenConfig generates the configuration structure definition file of the rpc service,
// which contains the zrpc.RpcServerConf configuration item by default.
// You can specify the naming style of the target file name through config.Config. For details,
// see https://github.com/r27153733/fastgozero/tree/master/tools/goctl/config/config.go
func (g *Generator) GenConfig(ctx DirContext, _ parser.Proto, cfg *conf.Config) error {
	dir := ctx.GetConfig()
	configFilename, err := format.FileNamingFormat(cfg.NamingFormat, "config")
	if err != nil {
		return err
	}

	fileName := filepath.Join(dir.Filename, configFilename+".go")
	if pathx.FileExists(fileName) {
		return nil
	}

	text, err := pathx.LoadTemplate(category, configTemplateFileFile, configTemplate)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, []byte(text), os.ModePerm)
}
