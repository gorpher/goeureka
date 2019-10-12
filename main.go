package main

import (
	"github.com/wx11055/goeureka/eureka"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {

	eureka.New(eureka.AppInfo{"goeureka", "127.0.0.1", 8080, "", ""}) // Performs eurekaClient registration
	go eureka.Client.Register()
	startWebServer() // Starts HTTP service  (async)

}

func startWebServer() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("hello world"))
	})
	log.Println("Starting HTTP service at 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("An error occured starting HTTP listener at port 8080")
		log.Println("Error: " + err.Error())
	}
}
