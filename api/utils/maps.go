package utils

func MapMap[K comparable, V any, R any](m map[K]V, f func(K, V) R) map[K]R {
	res := make(map[K]R, len(m))

	for k, v := range m {
		res[k] = f(k, v)
	}

	return res
}
