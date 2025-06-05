//go:generate go run github.com/lukasngl/godogen/cmd/godogen
//nolint:unused
package godogs

import (
	"context"
)

//godogen:given correct (number) of groups
//godogen:given (too) (many) groups
//godogen:given not enough of groups
func paramGroupMismatch(ctx context.Context, number string) error {
	return nil
}

//godogen:given wrong (parameter) type
func wrongParameterType(ctx context.Context, dogs rune) error {
	return nil
}

//godogen:then we have a wrong return type
func wrongReturnType() (string, error) {
	return "", nil
}

//godogen:step )malformed pattern(
func malformedPattern(ctx context.Context) error {
	return nil
}
