# go-resthooks

[REST Hooks](http://resthooks.org) library for Golang.

### Features

- Subscribe/Unsubscribe endpoints
- Exponential retries
- Decoupled subscription storage

### Todo

- Implement, list, get and update subscription endpoints

### Usage

```Go
import (
    "net/http"

    resthooks "github.com/marconi/go-resthooks"
)

func main() {
    ...
    rh := resthooks.NewResthook(store)
    defer rh.Close()

    http.Handle("/hooks/", rh.Handler())
}
```

Where `store` is a struct that implements the [ResthookStore](https://godoc.org/github.com/marconi/go-resthooks#ResthookStore) interface.

Its up to you where or how you would store the subscription, as long as you implement the interface and pass the data access object. This gives you flexibility to choose your favorite database and ORM.

Also `rh.Handler()` returns an `http.Handler` so you can mount it on prefix and any route that supports that standard interface. You can even wrap it with say authentication middleware, all 3rd-party middlewares that support the standard intefface `http.Handle` should work. Here we are just using the built-in `http.Handle`. In this setup, we'll get the following routes:

1. POST /hooks/subscribe
2. DELETE /hooks/unsubscribe

The first one is used to create a subscription and the second one to delete the subscription. Again how you handle this is up to your `ResthookStore` implementation, for example you can choose soft-delete when deleting subscription for auditing later.

Once you have a subscription, you can notify the subscribers with:

```Go
userId := 1
event := "post_created"
data := new(SampleData)

if err := rh.Notify(userId, event, data); err != nil {
  // handle error
}
```

If you want to get notified with what happened to the notifications, you subscribe to the results channel with:

```Go
go func() {
  for data := range rh.GetResults() {
    // handle data
  }
}()
```

Here `data` is an instance of [Notification](https://godoc.org/github.com/marconi/go-resthooks#Notification) so you can use the `Status` field if it was successful or not. Note that by default if a notification fails, it'll retry it 3 more times and only then will it give-up. The retry works as follows:

> // start retrying after 5 seconds and  
> // grow exponentially after that:  
> // 1st retry = after 5 seconds.  
> // 2nd retry = 5 * 3 = after 15 seconds.   
> // 3rd retry (final) = 5 * 3 * 3 = after 45 seconds. 

See https://godoc.org/github.com/marconi/go-resthooks#pkg-constants

Which you can configure with [Config](https://godoc.org/github.com/marconi/go-resthooks#Config).
