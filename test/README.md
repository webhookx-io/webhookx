# Integration Tests


1. starts httpbin

```
docker run -p 9999:80 kennethreitz/httpbin
```

2. runs integration tests

```
make test-integration
```
