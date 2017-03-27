package main

import (
	"log"

	"github.com/teambition/snapper-producer-go/snapper"
)

var (
	options = &snapper.Options{
		SecretKeys: []string{"Securekey"},
		ExpiresIn:  604800,
		Address:    "127.0.0.1:7720",
		ProducerID: "teambition",
	}
)

func main() {

	producer := snapper.New(options)
	producer.OnReconnect = func(err error) {
		log.Println("onReconnect:", err)
	}

	producer.OnError = func(err error) {
		log.Println("onError:", err)
	}
	err := producer.Connect()
	if err != nil {
		log.Println(err)
	} else {
		sss := "{\"e\":\":change:user/58b7ab90b7162c96120cba9b\",\"d\":{\"ated\":0,\"normal\":10,\"later\":0,\"private\":1,\"badge\":10,\"hasNormal\":true,\"hasAted\":false,\"hasLater\":false,\"hasPrivate\":true}}"
		producer.JoinRoom("user58b7ab90b7162c96120cba9b", sss)
	}
}
