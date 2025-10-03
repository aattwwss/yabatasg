package ltaapi

type Response[T any] struct {
	Value []T `json:"value"`
}
