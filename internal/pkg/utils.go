package pkg

import "fmt"

func ConvertInterfaceSliceToXSlice[T comparable](slice []interface{}) ([]T, bool) {
	var assertedSlice []T = []T{}
	for _, value := range slice {
		assertedValue, ok := value.(T)
		if !ok {
			fmt.Println("Type assertion of element to given type failed")
			return nil, false
		}
		assertedSlice = append(assertedSlice, assertedValue)
	}

	return assertedSlice, true
}

func RemoveByValue[T comparable](slice []T, value T) []T {
	for i, v := range slice {
		if v == value {
			return append(slice[:i], slice[i+1:]...)
		}
	}

	return slice
}
