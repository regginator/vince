package util

func SliceDel(arr []any, index int) []any {
	return append(arr[:index], arr[index+1:]...)
}
