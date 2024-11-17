package collections

import "slices"

func FindIndex[T comparable](slice []T, callback func(T, int) bool) int {
	for idx, element := range slice {
		if callback(element, idx) {
			return idx
		}
	}

	return -1
}

func Remove[T comparable](slice []T, callback func(T, int) bool) []T {
	idx := FindIndex(slice, callback)
	if idx == -1 {
		return slice
	}

	return slices.Delete(slice, idx, idx+1)
}
