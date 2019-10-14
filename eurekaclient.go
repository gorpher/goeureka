package goeureka

import (
	"encoding/json"
	"fmt"
	"github.com/wx11055/goeureka/model"
	"github.com/wx11055/gone/logger"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type eurekaClient struct {
	AppInfo  AppInfo
	instance model.Instance
}
type HttpRequest interface {
	DoRequest() ([]byte, error)
}

// 公共请求数据
type HttpAction struct {
	Method      string `yaml:"method"`
	Url         string `yaml:"url"`
	Body        string `yaml:"body"`
	Template    string `yaml:"template"`
	Accept      string `yaml:"accept"`
	ContentType string `yaml:"contentType"`
	Title       string `yaml:"title"`
	StoreCookie string `yaml:"storeCookie"`
	HandlerFunc func() ([]byte, error)
}

type AppInfo struct {
	AppID      string
	Host       string
	Port       int
	InstanceID string
	EurekaURL  string
}

var Client *eurekaClient
var defaultUrl = "http://localhost:8761" //默认注册中心地址

func init() {
	Client = new(eurekaClient)
	Client.instance = model.Instance{
		InstanceID:       "",
		App:              "",
		HostName:         "",
		IpAddr:           "",
		Sid:              "na",
		HomePageUrl:      "",
		StatusPageUrl:    "",
		HealthCheckUrl:   "",
		VipAddress:       "",
		SecureVipAddress: "",
		CountryId:        1,
		DataCenterInfo: model.DataCenterInfo{
			Class: "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
			Name:  "MyOwn",
		},
		Status:           model.DOWN,
		Overriddenstatus: model.DOWN,
		LeaseInfo: model.LeaseInfo{
			RenewalIntervalInSecs: 30,
			DurationInSecs:        90,
		},
		IsCoordinatingDiscoveryServer: false,
		LastUpdatedTimestamp:          0,
		LastDirtyTimestamp:            0,
		ActionType:                    model.ADDED,
		Port: model.Port{
			Enabled: true,
			Value:   7001,
		},
		SecurePort: model.Port{
			Enabled: false,
			Value:   7002,
		},
	}
}

func New(appInfo AppInfo) {
	Client.AppInfo = appInfo
	if Client.AppInfo.InstanceID == "" {
		Client.AppInfo.InstanceID = Client.AppInfo.Host + ":" + Client.AppInfo.AppID + ":" + strconv.Itoa(Client.AppInfo.Port)
	}
	if Client.AppInfo.EurekaURL == "" {
		Client.AppInfo.EurekaURL = defaultUrl
	}
	Client.instance.InstanceID = Client.AppInfo.InstanceID
	Client.instance.App = Client.AppInfo.AppID
	Client.instance.HostName = Client.AppInfo.Host
	Client.instance.Status = model.UP
	Client.instance.Overriddenstatus = model.UP
	Client.instance.Port.Value = Client.AppInfo.Port
	Client.instance.SecurePort.Value = Client.AppInfo.Port + 10
	Client.instance.HomePageUrl = "http://" + Client.instance.HostName + ":" + strconv.Itoa(Client.instance.Port.Value) + "/"
	Client.instance.StatusPageUrl = "http://" + Client.instance.HostName + ":" + strconv.Itoa(Client.instance.Port.Value) + "/info"
	Client.instance.HealthCheckUrl = "http://" + Client.instance.HostName + ":" + strconv.Itoa(Client.instance.Port.Value) + "/health"
	Client.instance.VipAddress = Client.instance.HostName
	Client.instance.SecureVipAddress = Client.instance.HostName

}

func handleSigterm() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		Client.Deregister()
		os.Exit(1)
	}()
}

// 注册上线
// POST {{url}}/eureka/apps/{{appID}}
func (e *eurekaClient) Register() {
	inst := struct {
		Instance model.Instance `json:"instance"`
	}{
		Client.instance,
	}

	data, err := json.Marshal(inst)
	if err != nil {
		log.Println(" instance mode to json err:", err)
		return
	}
	//logger.Debug(string(data))
	action := HttpAction{
		Url:         Client.AppInfo.EurekaURL + "/eureka/apps/" + Client.instance.App,
		Method:      "POST",
		ContentType: "application/json",
		Body:        string(data),
	}

	var result bool
	// 每隔5秒尝试注册到eureka,直到注册成功.
	for {
		logger.Debug("Trying to Register to Eureka Server...")
		result = action.DoRequest(nil)
		if result {
			logger.Debug("Register success...")
			break
		} else {
			time.Sleep(time.Second * 5)
		}
	}

	handleSigterm()     // Graceful shutdown on Ctrl+C or kill
	go StartHeartbeat() // Performs eurekaClient heartbeating (async)
	// Block...
	wg := sync.WaitGroup{} // Use a WaitGroup to block main() exit
	wg.Add(1)
	wg.Wait()
}

// 发送心跳
// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}
func StartHeartbeat() {
	// 心跳请求函数
	var heartbeat = func() {
		action := HttpAction{
			Url:    Client.AppInfo.EurekaURL + "/eureka/apps/" + Client.instance.App + "/" + Client.instance.InstanceID,
			Method: "PUT",
		}
		logger.Debug("Trying to send instance heart beat...")
		res := action.DoRequest(nil)
		if res {
			logger.Debug("send instance heart beat success!")
			return
		}
		logger.Error("send instance heart beat failed!!!")

	}
	// 每隔30秒发送一次心跳
	for {
		time.Sleep(time.Second * 30)
		heartbeat()
	}
}

// 下线
// DELETE {{url}}/eureka/apps/{{appID}}/{{instanceID}}
func (e *eurekaClient) Deregister() {
	logger.Info("Trying to deregister application...")
	// Deregister
	action := HttpAction{
		Url:    Client.AppInfo.EurekaURL + "/eureka/apps/" + Client.instance.App + "/" + Client.instance.InstanceID,
		Method: "DELETE",
	}
	action.DoRequest(nil)
	logger.Info("Deregistered application, exiting. Check eurekaClient...")
}

// 获取所有实例
// GET {{url}}/eureka/apps
func (e *eurekaClient) GetApps() model.Applications {
	fmt.Println("Trying to query for all appID instances...")
	action := HttpAction{
		Url:    Client.AppInfo.EurekaURL + "/eureka/apps",
		Method: "GET",
	}
	data := new(model.Applications)
	res := action.DoRequest(data)
	if res {
		return *data
	}
	logger.Error("get applications err")
	return *data
}

// 查询服务的所有实例
// GET {{url}}/eureka/apps/{{appID}}
func (e *eurekaClient) GetAppInstances() {
	fmt.Println("Trying to query for a specific appID/instanceID...")
	action := HttpAction{
		Url:    Client.AppInfo.EurekaURL + "/eureka/apps" + Client.instance.App,
		Method: "GET",
	}
	data := new(model.Application)
	res := action.DoRequest(data)
	if res {
		fmt.Println(data)
	}
}

// 查询指定服务实例
// GET {{url}}/eureka/apps/{{appID}}/{{appID}}:{{instanceID}}
func (e *eurekaClient) GetAppInstanceByID(url string, instance model.Instance) {
	fmt.Println("Trying to request all application instance detail...")
	action := HttpAction{
		Url:    url + "/eureka/apps" + instance.App + "/" + instance.InstanceID,
		Method: "GET",
	}
	data := new(model.Application)
	res := action.DoRequest(data)
	if res {
		fmt.Println(data)
	}
}

//更改实例状态
// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}/status?value=OUT_OF_SERVICE
func (e *eurekaClient) UpdateAppInstanceStatus() {
	fmt.Println("Trying to take instance out of service ...")
	action := HttpAction{
		Url:    Client.AppInfo.EurekaURL + "/eureka/apps/" + Client.instance.App + "/" + Client.instance.InstanceID + "/status?value=" + model.OUT_OF_SERVICE,
		Method: "GET",
	}
	data := new(model.Application)
	res := action.DoRequest(data)
	if res {
		fmt.Println(data)
	}
}

//更新元数据
//{{url}}/eureka/v2/apps/{{appID}}/{{instanceID}}/metadata?key=value
func (e *eurekaClient) UpdateAppInstanceMetadata(vk string) {
	fmt.Println("Trying to update metadata ...")

	action := HttpAction{
		Url:    Client.AppInfo.EurekaURL + "/eureka/apps/" + Client.instance.App + "/" + Client.instance.InstanceID + "/status?metadata?" + vk,
		Method: "PUT",
	}

	res := action.DoRequest(nil)
	if res {
		fmt.Println(res)
	}
}

// 查询vip下的实例
//GET {{url}}/eureka/vips/{{vipAddress}}
func (e *eurekaClient) GetAppInstanceVip() {
	fmt.Println("Trying to query instance in the a svip ...")
	action := HttpAction{
		Url:    Client.AppInfo.EurekaURL + "/eureka/vips/" + Client.instance.VipAddress,
		Method: "GET",
	}
	data := new(model.Applications)
	res := action.DoRequest(data)
	if res {
		fmt.Println(data)
	}
}

// 查询svip下的实例
//GET {{url}}/eureka/svips/{{svipAddress}}
func (e *eurekaClient) GetAppInstanceSVip() {
	fmt.Println("Trying to query instance in the a svip ...")
	action := HttpAction{
		Url:    Client.AppInfo.EurekaURL + "/eureka/svips/" + Client.instance.SecureVipAddress,
		Method: "GET",
	}
	data := new(model.Applications)
	res := action.DoRequest(data)
	if res {
		fmt.Println(data)
	}
}

var ServiceCache map[string]model.Instance

// 通过自己定义请求其他服务
func (e *eurekaClient) SendRawRequest(serviceName string, path string, method string, body io.Reader) {
	instance := ServiceCache[serviceName]
	if instance.HomePageUrl != "" {
		// todo 服务之间调用
	}
}

// 通过request装饰器请求其他服务
func (e *eurekaClient) Request(req *http.Request) (*http.Response, error) {
	client := &http.Client{Transport: &http.Transport{}}
	return client.Do(req)
}
