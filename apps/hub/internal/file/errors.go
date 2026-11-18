package file

// FieldError reports one member of the start of an upload that the file kind refuses.
type FieldError struct {
	Pointer string
	Detail  string
}

// Error returns the member and the reason.
func (e *FieldError) Error() string {
	return e.Pointer + ": " + e.Detail
}
