package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorpher/goeureka"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
)

var client *goeureka.Client

func main() {
	app := flag.String("app", "golang-service-1", "服务名,默认值是golang-service")
	port := flag.Int("port", 8080, "端口,默认值是8080")
	flag.Parse()
	var err error
	client, err = goeureka.New(&goeureka.AppInfo{
		AppID:     *app,
		Port:      *port,
		UserName:  "fitmgr",
		Password:  "fitmgr",
		EurekaURL: "http://127.0.0.1:8761",
	}) // Performs eurekaClient registration
	if err != nil {
		panic(err)
	}
	err = client.Register()
	if err != nil {
		panic(err)
	}

	go func() {
		ch := makeShutdownCh()
		select {
		case <-ch:
			client.Deregister()
		}
	}()
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
	http.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		m := map[string]string{}
		request, err := http.NewRequest(http.MethodGet, "http://golang-service/info", nil)
		if err != nil {
			panic(err)
		}
		resp, err := client.Do(request)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer resp.Body.Close() //nolint
		all, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		m["call"] = string(all)
		m["desc"] = "调用其他服务成功"
		v, _ := json.Marshal(m)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(v)
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
	log.Printf("Starting HTTP service at %d \n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Printf("An error occurred starting HTTP listener at port %d \n", port)
		log.Print("Error: " + err.Error())
	}
}
