package session

import (
	"context"

	"firebase.google.com/go/v4/auth"
)

// TokenFromContext retrieves the verified Firebase ID token stored by authentication middleware.
// The second return value reports whether the token was present.
func TokenFromContext(ctx context.Context) (*auth.Token, bool) {
	if ctx == nil {
		return nil, false
	}
	tok, ok := ctx.Value(IdTokenKey).(*auth.Token)
	return tok, ok && tok != nil
}
