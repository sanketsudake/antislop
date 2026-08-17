package nosources

import "context"

type key struct{}

// With sources set to the empty list every narrowing of any is reported.
func fromContext(ctx context.Context) (string, bool) {
	ns, ok := ctx.Value(key{}).(string) // want `nonarrowany: comma-ok assertion on any`
	return ns, ok
}
