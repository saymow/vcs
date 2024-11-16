package collections

func FindIndex[T comparable](slice []T, callback func(T, int) bool) int {
	for idx, element := range slice {
		if callback(element, idx) {
			return idx
		}
	}

	return -1
}

func ToMap[T any, G comparable](slice []T, keyExtractor func(T, int) G) map[G]T {
	myMap := make(map[G]T)

	for idx, element := range slice {
		myMap[keyExtractor(element, idx)] = element
	}

	return myMap
}
