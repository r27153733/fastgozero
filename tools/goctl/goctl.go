package main

import (
	"github.com/r27153733/fastgozero/core/load"
	"github.com/r27153733/fastgozero/core/logx"
	"github.com/r27153733/fastgozero/tools/fastgoctl/cmd"
)

func main() {
	logx.Disable()
	load.Disable()
	cmd.Execute()
}
