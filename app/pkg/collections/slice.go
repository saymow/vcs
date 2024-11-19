package collections

import "slices"

func Map[T any, F any](slice []T, callback func(T, int) F) []F {
	newSlice := make([]F, len(slice), cap(slice))

	for idx, element := range slice {
		newSlice[idx] = callback(element, idx)
	}

	return newSlice
}

func Filter[T any](slice []T, callback func(T, int) bool) []T {
	newSlice := []T{}

	for idx, element := range slice {
		if callback(element, idx) {
			newSlice = append(newSlice, element)
		}
	}

	return newSlice
}

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
