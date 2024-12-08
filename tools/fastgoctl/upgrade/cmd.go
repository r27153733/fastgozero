package upgrade

import "github.com/r27153733/fastgozero/tools/fastgoctl/internal/cobrax"

// Cmd describes an upgrade command.
var Cmd = cobrax.NewCommand("upgrade", cobrax.WithRunE(upgrade))
