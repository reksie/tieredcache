## Improvements

- in the case we have multiple calls we can try to use something like [singleflight](https://pkg.go.dev/golang.org/x/sync@v0.8.0/singleflight) to try and dedupe.
- however within a request unless there's multiple paralell calls, we will typically be waiting when there is a cache miss, and future requests will get cache hits.
