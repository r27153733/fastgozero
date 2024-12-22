//go:build gozerorouter

package pathvar

var pathVars = contextKey("pathVars")

type contextKey string

func (c contextKey) String() string {
	return "rest/router/pathvar/context key: " + string(c)
}
