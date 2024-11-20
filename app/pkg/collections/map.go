package collections

func InvertMap[T comparable, S comparable](receivedMap map[T]S) map[S][]T {
	responseMap := make(map[S][]T)

	for key, value := range receivedMap {
		if _, ok := responseMap[value]; !ok {
			responseMap[value] = []T{}
		}

		responseMap[value] = append(responseMap[value], key)
	}

	return responseMap
}
