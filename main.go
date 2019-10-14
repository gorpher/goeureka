package main

import (
	"github.com/wx11055/goeureka/eureka"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
)

func main() {
	port := 9090
	eureka.New(eureka.AppInfo{AppID: "goeureka", Host: "127.0.0.1", Port: port}) // Performs eurekaClient registration
	go eureka.Client.Register()
	startWebServer(port) // Starts HTTP service  (async)

}

var healthData = `
{
"description": "Golang Eureka Discovery Client",
"status": "UP"
}
`

func startWebServer(port int) {
	http.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(healthData))
	})
	log.Println("Starting HTTP service at 8080")
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		log.Println("An error occured starting HTTP listener at port 8080")
		log.Println("Error: " + err.Error())
	}
}
