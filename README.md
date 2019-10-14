[![Build Status](https://secure.travis-ci.org/wx11055/travis.png?branch=master)](http://travis-ci.org/wx11055/goeureka)
[![Postman Test](https://res.cloudinary.com/postman/image/upload/w_152,h_56,c_fit,f_auto,t_team_logo/v1/team/768118b36f06c94b0306958b980558e6915839447e859fe16906e29d683976f0)](https://documenter.getpostman.com/view/3593333/SVtVVonB?version=latest)
# goeureka
Goland Eureka Client for Spring Cloud Eureka 1.x



# Usage

```go
package main

import (
	"github.com/wx11055/goeureka"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
)

func main() {
	port := 9090
	goeureka.New(goeureka.AppInfo{AppID: "goeureka", Host: "127.0.0.1", Port: port}) // Performs eurekaClient registration
	go goeureka.Client.Register()
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
```

# reference
- [Eureka REST operations](https://github.com/Netflix/eureka/wiki/Eureka-REST-operations)
- [gopkg.in](https://github.com/h2non/gock)
- [A Tiny Test Framework for Go](https://github.com/nbio/st)
