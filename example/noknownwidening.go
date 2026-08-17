package example

// Rejected by noknownwidening.
var retries any = 3

// Accepted: the declaration keeps what the compiler already knew.
var retryCount = 3

// Accepted: forwarding a value to an API that asks for any is not the same as
// forgetting its type.
func LogRetries(log func(args ...any)) { log(any(retryCount)) }
