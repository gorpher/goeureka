package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorpher/goeureka"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
)

var client *goeureka.Client

func main() {
	app := flag.String("app", "golang-service-echo", "服务名,默认值是golang-service-echo")
	port := flag.Int("port", 9999, "端口,默认值是8080")
	username := flag.String("username", "fitmgr", "用户名,默认值是fitmgr")
	password := flag.String("password", "fitmgr", "密码,默认值是fitmgr")
	eurekaurl := flag.String("eurekaurl", "http://127.0.0.1:8761", "eureka链接地址,默认值是http://127.0.0.1:8761")
	flag.Parse()
	var err error
	client, err = goeureka.New(&goeureka.AppInfo{
		AppID:     *app,
		Port:      *port,
		UserName:  *username,
		Password:  *password,
		EurekaURL: *eurekaurl,
	}) // Performs eurekaClient registration
	if err != nil {
		panic(err)
	}
	err = client.Register()
	if err != nil {
		panic(err)
	}

	startWebServer(*port) // Starts HTTP service  (async)
}

var ignoreSignals = []os.Signal{os.Interrupt}
var forwardSignals []os.Signal

// makeShutdownCh creates an interrupt listener and returns a channel.
// A message will be sent on the channel for every interrupt received.
func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, ignoreSignals...)
	signal.Notify(signalCh, forwardSignals...)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}

var healthData = `
{
"description": "Golang Eureka Discovery EurekaClient",
"status": "UP"
}
`

func startWebServer(port int) {
	http.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(healthData)) //nolint
	})
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		for i := range r.Header {
			w.Header().Add(i, r.Header.Get(i))
		}
		w.WriteHeader(http.StatusOK)
		io.Copy(w, r.Body) //nolint
		defer r.Body.Close()
	})
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		ip := goeureka.GetLocalIP()
		m := map[string]string{
			"HostName":   client.AppInfo.HostName,
			"Port":       strconv.Itoa(client.AppInfo.Port),
			"InstanceID": client.AppInfo.InstanceID,
			"AppID":      client.AppInfo.AppID,
			"IP":         ip,
			"IpAddr":     client.Instance.IpAddr,
		}
		v, _ := json.Marshal(m)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(v)
	})
	go func() {
		log.Printf("Starting HTTP service at %d \n", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Printf("An error occurred starting HTTP listener at port %d \n", port)
			log.Print("Error: " + err.Error())
		}
	}()
	ch := makeShutdownCh()
	select {
	case <-ch:
		client.Deregister()
	}
}
