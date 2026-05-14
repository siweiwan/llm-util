package app

type App interface {
	Id() string
	Key() string
	SendRequest(request interface{}) (interface{}, error)
}
