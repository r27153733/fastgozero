package upgrade

import (
	"fmt"
	"runtime"

	"github.com/r27153733/fastgozero/tools/fastgoctl/rpc/execx"
	"github.com/spf13/cobra"
)

// upgrade gets the latest goctl by
// go install github.com/r27153733/fastgozero/tools/goctl@latest
func upgrade(_ *cobra.Command, _ []string) error {
	cmd := `GO111MODULE=on GOPROXY=https://goproxy.cn/,direct go install github.com/r27153733/fastgozero/tools/goctl@latest`
	if runtime.GOOS == "windows" {
		cmd = `set GOPROXY=https://goproxy.cn,direct && go install github.com/r27153733/fastgozero/tools/goctl@latest`
	}
	info, err := execx.Run(cmd, "")
	if err != nil {
		return err
	}

	fmt.Print(info)
	return nil
}
