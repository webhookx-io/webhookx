package router

type Route struct {
	Paths   []string
	Methods []string
	Handler interface{}
}
