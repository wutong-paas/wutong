package bcode

// tenant env 11300~11399
var (
	ErrNamespaceExists = newByMessage(400, 11300, "tenant env namespace exists")
)
