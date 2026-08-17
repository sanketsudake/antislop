package nosources

import "context"

type key struct{}

// With sources set to the empty list every unchecked assertion is reported.
func fromContext(ctx context.Context) string {
	return ctx.Value(key{}).(string) // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}
