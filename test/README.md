# Integration Tests


1. starts dependencies

```shell
make deps
```

2. runs integration tests

```shell
make test-integration
```


# How-To

#### How to run specific tests?

`FIt`, `FContext`.

```
Context("some specs you're debugging", func() {
  It("might be failing", func() { ... })
  FIt("might also be failing", func() { ... })
})

```

```
ginkgo
```

This will run only test "might also be failing", skip the rest.

See https://onsi.github.io/ginkgo/#focused-specs.
