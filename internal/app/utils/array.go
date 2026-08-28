package utils

// UnpackArray принимает слайс любого типа ([]int, []string, []Person и т. д.) и преобразует его в []any.
func UnpackArray[S ~[]E, E any](s S) []any {
	r := make([]any, len(s))
	for i, e := range s {
		r[i] = e
	}
	return r
}
