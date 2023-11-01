package util

func Ptr[T any](v T) *T {
	return &v
}

func Value[T any](p *T) T {
	return *p
}
