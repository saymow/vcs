package collections

func FindIndex[T comparable](slice []T, callback func(T, int) bool) int {
	for idx, element := range slice {
		if callback(element, idx) {
			return idx
		}
	}

	return -1
}
