package example

import "reflect"

// Rejected by noreflect. User is declared in noanyparams.go.
func ID(u User) string { return reflect.ValueOf(u).FieldByName("ID").String() }

// Accepted: the field is named in the type, so the compiler checks the access.
func UserID(u User) string { return u.ID }
