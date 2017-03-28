# snapper-producer-go

[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/teambition/snapper-producer-go/master/LICENSE)
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/teambition/snapper-producer-go)

Snapper producer client for golang

## Example
```go
options = &snapper.Options{
    SecretKeys: []string{"tokenXXXXXXX"},
    ExpiresIn:  7800,
    Address:    "127.0.0.1:7720", 
    ProducerID: "testProducerId",
}
producer, _ := snapper.New(options)
producer.OnReconnect = func(err error) {
  log.Println("onReconnect:", err)
}
producer.OnError = func(err error) {
  log.Println("onError:", err)
}
err := producer.Connect()

// generate a token for a consumer
var token = producer.SignAuth("userIdxxx")

// send a message to a room
producer.SendMessage("room","message")
producer.SendMessage("project58b7ad3b5b2387b92fc68faf","message")
producer.sendMessage("projects51762b8f78cfa9f357000011", "{"e":":remove:tasks","d":"553f569aca14974c"}")

producer.JoinRoom("projects51762b8f78cfa9f357000011', 'lkoH6jeg8ATcptZQFHHH7w~~")
producer.LeaveRoom("projects51762b8f78cfa9f357000011", "lkoH6jeg8ATcptZQFHHH7w~~")

producer.Request("method", "arguments")
producer.Request("consumers", string[]{"userIdxxx"})

producer.Close()
```   

## API

### snapper.Options
```go
  roducer, _ := snapper.New(&snapper.Options{})
```
- SecretKeys: A array of string or buffer containing either the secret for HMAC algorithms, or the PEM encoded private key for RSA and ECDSA.
- ExpiresIn: default to 3600 * 24, one day.
- ProducerID: String, producer's name, use for log.
- Address:  Snapper server address, default to '127.0.0.1:7720'.
- MessageWaitCount: int, default to 10000.
- ReadWriteTimeout: int, default to 60 seconds

###  producer.SignAuth(payload map[string]interface{})
Generate a token string. payload should have userId property that Snapper server determine which room the consumer should auto join.
