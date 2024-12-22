//go:build !gozerorouter

package pathvar

import "github.com/r27153733/fastgozero/rest/router/internal/httprouter"

var pathVars = httprouter.ParamsKey
