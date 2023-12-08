# Retry
 Simple retry library with exponential backoff and timeouts

## Get

`go get github.com/chk-n/retry`

 ## Example

```go
// initialise retry
r := retry.NewDefault()

err := r.Do(func() error {
 // Call error prone function
})

err = r.DoTimeout(time.Second, func() error {
 // Do network call with timeout
})
```
