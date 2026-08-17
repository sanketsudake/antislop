package example

// Rejected by noanyreturns.
func Load() any { return nil }

// Accepted: return the type the caller needs. User is declared in noanyparams.go.
func LoadUser() (User, error) { return User{}, nil }

// Accepted: (*IntHeap).Pop() any in noanyparams.go returns any because
// container/heap.Interface dictates that signature, so it is not reported.
