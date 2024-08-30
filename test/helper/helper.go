package helper

import "github.com/go-resty/resty/v2"

func AdminClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8080")
	return c
}

func ProxyClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8081")
	return c
}
