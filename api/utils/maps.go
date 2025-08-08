package utils

// MapMap maps a map to a new map using a function. Examples:
// - map[string]int => map[string][]byte
// - map[int]bool => map[int]string
// - map[string]*MyStruct => map[string]io.Reader
func MapMap[KEY comparable, A any, B any](m map[KEY]A, f func(KEY, A) B) map[KEY]B {
	res := make(map[KEY]B, len(m))

	for k, v := range m {
		res[k] = f(k, v)
	}

	return res
}
