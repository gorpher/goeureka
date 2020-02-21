package goeureka

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var log = zerolog.New(os.Stderr).With().Timestamp().Logger()

type serviceCache struct {
	v map[string]map[int64]string
	sync.RWMutex
	timeout time.Duration
}

// 开启协程,清除缓存
func NewServiceCache(timeout time.Duration) *serviceCache {
	cache := &serviceCache{v: map[string]map[int64]string{}, timeout: timeout}
	go func() {
		for {
			time.Sleep(timeout / 3) //根据超时时间设置1/3的收集
			if len(cache.v) > 0 {
				for k := range cache.v {
					for i := range cache.v[k] {
						cache.Lock()
						if i > time.Now().UnixNano() {
							delete(cache.v[k], i)
						}
						cache.Unlock()
					}
				}
			}
		}
	}()
	return cache
}

// 获取时间key
func (s *serviceCache) Set(k, v string) int64 {
	s.Lock()
	defer s.Unlock()
	key := time.Now().Add(s.timeout).UnixNano()
	k = strings.ToUpper(k) //全部大写
	if s.v[k] == nil {
		s.v[k] = map[int64]string{
			key: v,
		}
	} else {
		s.v[k][key] = v
	}

	return key
}

// 根据服务名,获取第一个服务实例信息
func (s *serviceCache) GetFirst(k string) string {
	s.RLock()
	defer s.RUnlock()
	k = strings.ToUpper(k)
	if s.v[k] != nil && len(s.v[k]) > 0 {
		key := int64(math.MaxInt64)
		for i := range s.v[k] {
			if i < key {
				key = i
			}
		}
		return s.v[k][key]
	}
	return ""
}

// 根据key,获取最后一个服务实例
func (s *serviceCache) GetLast(k string) string {
	s.RLock()
	defer s.RUnlock()
	k = strings.ToUpper(k)
	if s.v[k] != nil && len(s.v[k]) > 0 {
		key := int64(0)
		for i := range s.v[k] {
			if i > key {
				key = i
			}
		}
		return s.v[k][key]
	}
	return ""
}

// 获取服务实例
func (s *serviceCache) Get(k string) string {
	s.RLock()
	defer s.RUnlock()
	k = strings.ToUpper(k)
	if s.v[k] != nil && len(s.v[k]) > 0 {
		for i := range s.v[k] {
			return s.v[k][i]
		}
	}
	return ""
}

// 删除服务实例
func (s *serviceCache) Delete(k string) {
	s.Lock()
	defer s.Unlock()
	if len(s.v) > 0 {
		delete(s.v, k)
	}
}

// eureka客户端
type Client struct {
	AppInfo         *AppInfo
	Instance        Instance
	serviceCache    *serviceCache
	heartbeatCh     chan struct{} // 心跳通道
	stopChan        chan struct{} //停止发送心跳
	stopChanDone    chan struct{} //停止发送心跳成功
	originTransport http.RoundTripper
	log             zerolog.Logger
}

var defaultUrl = "http://localhost:8761" //默认注册中心地址

// 新建goeureka客户端
func New(appInfo *AppInfo) (*Client, error) {
	log = log.Level(appInfo.LogLevel)
	log = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02 15:04:05",
	})
	// 设置日志等级
	if appInfo.Port == 0 {
		return nil, fmt.Errorf("端口port不能为空")
	}
	if appInfo.AppID == "" {
		return nil, fmt.Errorf("appID不能为空")
	}
	if appInfo.EurekaURL == "" {
		return nil, fmt.Errorf("EurekaURL不能为空")
	}
	_, err := url.Parse(defaultUrl)
	if err != nil {
		return nil, fmt.Errorf("EurekaURL不合法: %v", err)
	}
	c := &Client{
		AppInfo: appInfo,
		Instance: Instance{
			Sid:       "na",
			CountryId: 1,
			DataCenterInfo: DataCenterInfo{
				Class: "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
				Name:  "MyOwn",
			},
			Status:           DOWN,
			Overriddenstatus: DOWN,
			LeaseInfo: LeaseInfo{
				RenewalIntervalInSecs: 30,
				DurationInSecs:        90,
			},
			IsCoordinatingDiscoveryServer: false,
			LastUpdatedTimestamp:          0,
			LastDirtyTimestamp:            0,
			ActionType:                    ADDED,
			Port: Port{
				Enabled: true,
				Value:   7001,
			},
			SecurePort: Port{
				Enabled: false,
				Value:   7002,
			},
		},
		stopChan:        make(chan struct{}, 1),
		stopChanDone:    make(chan struct{}, 1),
		heartbeatCh:     make(chan struct{}, 1),
		serviceCache:    NewServiceCache(time.Second * 5), //新建缓存器
		originTransport: http.DefaultTransport,
	}
	hostname, _ := os.Hostname() // nolint
	//查找当前hostname
	if c.AppInfo.HostName == "" && hostname == "" {
		c.AppInfo.HostName = hostname
	}
	if c.AppInfo.InstanceID == "" {
		c.AppInfo.InstanceID = hostname + ":" + c.AppInfo.AppID + ":" + strconv.Itoa(c.AppInfo.Port)
	}
	if c.AppInfo.EurekaURL == "" {
		c.AppInfo.EurekaURL = defaultUrl
	}
	c.Instance.InstanceID = c.AppInfo.InstanceID
	c.Instance.App = c.AppInfo.AppID
	c.Instance.HostName = c.AppInfo.HostName
	ips := GetLocalIPS()
	if len(ips) > 0 {
		c.Instance.IpAddr = ips[0]
	}
	c.Instance.VipAddress = c.AppInfo.AppID
	c.Instance.SecureVipAddress = c.AppInfo.AppID
	c.Instance.Status = UP
	c.Instance.Overriddenstatus = UP
	c.Instance.Port.Value = c.AppInfo.Port
	c.Instance.SecurePort.Value = 443

	c.Instance.HomePageUrl = "http://" + c.Instance.HostName + ":" + strconv.Itoa(c.Instance.Port.Value) + "/"
	c.Instance.StatusPageUrl = "http://" + c.Instance.HostName + ":" + strconv.Itoa(c.Instance.Port.Value) + "/info"
	c.Instance.HealthCheckUrl = "http://" + c.Instance.HostName + ":" + strconv.Itoa(c.Instance.Port.Value) + "/health"

	return c, nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func randomStr(num int) string {
	seed := time.Now().UnixNano()
	if seed <= 0 {
		seed = time.Now().UnixNano()
	}
	rand.Seed(seed)
	b := make([]rune, num)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// 注册上线
// POST {{url}}/eureka/apps/{{appID}}
func (c *Client) Register() error {
	inst := struct {
		Instance Instance `json:"instance"`
	}{
		c.Instance,
	}
	data, err := json.Marshal(inst)
	if err != nil {
		log.Debug().AnErr("注册服务实例失败:序列化实例信息失败", err)
		return errors.Wrap(err, "注册服务实例失败")
	}
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App,
		Method:      "POST",
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
		Body:        string(data),
	}

	err = doWithRetry("Register Eureka Server", 3, 3*time.Second, func() error {
		log.Debug().Msgf("尝试注册到注册中心 EurekaURL=%s", c.AppInfo.EurekaURL)
		err = action.DoRequest(nil)
		if err == nil {
			log.Debug().Msgf("注册成功 EurekaURL=%s,AppID=%s,InstanceID=%s", c.AppInfo.EurekaURL, c.AppInfo.AppID, c.AppInfo.InstanceID)
			return nil
		}
		return err
	})
	if err != nil {
		log.Error().AnErr(fmt.Sprintf("注册服务实例失败 url=%s", action.Url), err)
		return errors.Wrap(err, "注册服务实例失败,请求eureka server失败")
	}
	// 刷新获取所有服务加入到缓存
	go c.refreshAllService()

	// 每隔5秒尝试注册到eureka,直到注册成功.
	go func() {
		c.enableHeartbeat() //开始发送心跳
	}()
	c.starHeartbeats()
	return err
}

// 发送心跳
func (c *Client) starHeartbeats() {
	//	启动协程接受停止消息和启动消息
	go func() {
		log.Debug().Msg("启动一个控制心跳协程,发送心跳命令.")
	endSend:
		for {
			select {
			case <-time.After(time.Second * 5):
				c.heartbeatCh <- struct{}{}
				log.Debug().Msg("五秒发送心跳一次 [完成]")
			case <-c.stopChan:
				log.Debug().Msg("接受到停止信号 停止发送心跳")
				break endSend
			}
		}
		c.stopChanDone <- struct{}{}
	}()
}

// 停止发送心跳
func (c *Client) stopHeartbeats() <-chan struct{} {
	if len(c.stopChan) == 0 {
		c.stopChan <- struct{}{}
	}
	return c.stopChanDone
}

// 刷新所有服务
func (c *Client) refreshAllService() {
	apps, err := c.GetApps()
	if err != nil {
		log.Error().AnErr("刷新所有服务失败", err)
		return
	}
	if len(apps.Application) > 0 {
		for i := range apps.Application {
			if len(apps.Application[i].Instance) > 0 {
				for k := range apps.Application[i].Instance {
					c.serviceCache.Set(apps.Application[i].Instance[k].App, fmt.Sprintf("%s:%d", apps.Application[i].Instance[k].HostName, apps.Application[i].Instance[k].Port.Value))
				}
			}
		}
	}
}

// 开启心跳
// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}
func (c *Client) enableHeartbeat() error {
	inst := struct {
		Instance Instance `json:"instance"`
	}{
		c.Instance,
	}
	data, err := json.Marshal(inst)
	if err != nil {
		log.Debug().AnErr("发送心跳失败,序列化请求参数失败", err)
		return err
	}
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App + "/" + c.Instance.InstanceID,
		Method:      "PUT",
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
		Body:        string(data),
	}
	// 心跳请求函数
	var heartbeat = func() error {
		log.Debug().Msgf("尝试发送心跳信息 InstanceID=%s", inst.Instance.InstanceID)
		err := action.DoRequest(nil)
		if err != nil {
			log.Error().AnErr("发送心跳信息失败", err)
			return err
		}
		log.Debug().Msg("发送心跳信息成功")
		return nil
	}

	//监听通道,如果有消息则发送心跳
	for range c.heartbeatCh {
		// 每隔30秒发送一次心跳
		err := heartbeat()
		if err != nil {
			log.Error().AnErr("发送心跳信息失败", err)
		}
	}
	return nil
}

// 下线
// DELETE {{url}}/eureka/apps/{{appID}}/{{instanceID}}
func (c *Client) Deregister() error {
	log.Debug().Msg("尝试从注册中心中注销,下线")
	// 停止发送心跳
	toStopHeartbeats := c.stopHeartbeats()
	select {
	case <-time.After(time.Second * 3):
		log.Warn().Msg("调用停止发送心跳函数,执行超时,开始下线")
	case <-toStopHeartbeats:
		log.Debug().Msgf("调用停止发送心跳函数,停止发送心跳 [成功]")
	}
	// Deregister
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App + "/" + c.Instance.InstanceID,
		Method:      http.MethodDelete,
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
	}
	log.Debug().Msgf("下线 URL=%s", action.Url)
	err := action.DoRequest(nil)
	if err != nil {
		log.Error().AnErr("从注册中心注销失败", err)
		return err
	}
	log.Debug().Msg("下线成功")
	return nil
}

// 获取所有实例
// GET {{url}}/eureka/apps
func (c *Client) GetApps() (Applications, error) {
	log.Debug().Msg("尝试查询获取所有实例")
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps",
		Method:      "GET",
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
	}
	data := new(Applications)
	err := action.DoRequest(data)
	if err != nil {
		log.Error().AnErr("获取所有实例失败", err)
		return Applications{}, err
	}

	return *data, err
}

// 获取指定服务的实例
// GET {{url}}/eureka/apps/{{appID}}
func (c *Client) GetAppInstances(appID string) (Application, error) {
	log.Debug().Msg("尝试获取指定服务实例")
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + appID,
		Method:      "GET",
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
	}
	data := new(Application)
	err := action.DoRequest(data)
	if err != nil {
		log.Error().AnErr("获取指定服务实例失败", err)
		return Application{}, err
	}
	return *data, err
}

// 查询同服务的所有实例
// GET {{url}}/eureka/apps/{{appID}}
func (c *Client) GetOwnAppInstances() (Application, error) {
	log.Debug().Msg("尝试查询指定服务的所有实例")
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App,
		Method:      "GET",
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
	}
	data := new(Application)
	err := action.DoRequest(data)
	if err != nil {
		log.Error().AnErr("查询指定服务的所有实例失败", err)
		return Application{}, err
	}
	return *data, err
}

// 查询自己注册的服务信息
// GET {{url}}/eureka/apps/{{appID}}/{{appID}}:{{instanceID}}
func (c *Client) GetOwnAppInstance() (Instance, error) {
	log.Debug().Msg("尝试查询自己注册的服务信息")
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App + "/" + c.Instance.InstanceID,
		Method:      "GET",
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
	}
	data := new(Instance)
	err := action.DoRequest(data)
	if err != nil {
		log.Error().AnErr("查询自己注册的服务信息失败", err)
		return Instance{}, err
	}
	return *data, err
}

// 查询指定服务实例
func (c *Client) GetAppInstanceByID(appID string, instanceID string) (Instance, error) {
	log.Debug().Msg("尝试查询指定服务实例信息")
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + appID + "/" + instanceID,
		Method:      http.MethodGet,
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
	}
	data := new(Instance)
	err := action.DoRequest(data)
	if err != nil {
		log.Error().AnErr("查询指定服务实例信息失败", err)
		return Instance{}, err
	}
	return *data, err
}

//更改自己实例状态
// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}/status?value=OUT_OF_SERVICE
func (c *Client) UpdateOwnAppInstanceStatus(status InstanceStatus) error {
	log.Debug().Msg("尝试更改自己实例状态")
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App + "/" + c.Instance.InstanceID + "/status?value=" + string(status),
		Method:      http.MethodPut,
		ContentType: "application/json",

		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}
	var doHeartBeat bool
	switch status {
	case DOWN:
		log.Debug().Msgf("更改状态下线 status=%s", DOWN)
		// 停止发送心跳
		toStopHeartbeats := c.stopHeartbeats()
		select {
		case <-time.After(time.Second * 3):
			log.Warn().Msgf("调用停止发送心跳函数,执行超时,开始更新状态 %s", DOWN)
		case <-toStopHeartbeats:
			log.Debug().Msgf("调用停止发送心跳函数,停止发送心跳 [成功]")
		}
	case OUT_OF_SERVICE:
		log.Debug().Msgf("更改状态下线 status=%s", OUT_OF_SERVICE)
		// 停止发送心跳
		toStopHeartbeats := c.stopHeartbeats()
		select {
		case <-time.After(time.Second * 3):
			log.Warn().Msgf("调用停止发送心跳函数,执行超时,开始更新状态 %s", OUT_OF_SERVICE)
		case <-toStopHeartbeats:
			log.Debug().Msgf("调用停止发送心跳函数,停止发送心跳 [成功]")
		}
	case UNKNOWN:
	case UP:
		c.startHeartBeat()
	case STARTING:
	}
	err := action.DoRequest(nil)
	if err == nil && doHeartBeat {
		log.Debug().Msg("重新开始发送心跳")
		c.startHeartBeat()
	}
	return err
}

//更新元数据
//{{url}}/eureka/v2/apps/{{appID}}/{{instanceID}}/metadata?key=value
func (c *Client) UpdateOwnAppInstanceMetadata(value map[string]string) error {
	log.Debug().Msg("尝试更新元数据")
	vk := url.Values{}
	for k := range value {
		vk.Add(k, value[k])
	}
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App + "/" + c.Instance.InstanceID + "/metadata?" + vk.Encode(),
		Method:      http.MethodPut,
		ContentType: "application/json",

		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}
	return action.DoRequest(nil)
}

// 查询vip下的实例
//GET {{url}}/eureka/vips/{{vipAddress}}
func (c *Client) GetAppInstanceVip(vip string) (Applications, error) {
	log.Debug().Msg("查询vip下的实例")
	action := httpAction{
		Url:      c.AppInfo.EurekaURL + "/eureka/vips/" + vip,
		Method:   http.MethodGet,
		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}
	data := new(Applications)
	err := action.DoRequest(data)
	if err != nil {
		log.Error().AnErr("查询vip下的实例失败", err)
		return Applications{}, err
	}
	return *data, err
}

// 查询svip下的实例
//GET {{url}}/eureka/svips/{{svipAddress}}
func (c *Client) GetAppInstanceSVip(svips string) (Applications, error) {
	log.Debug().Msg("查询svip下的实例")
	action := httpAction{
		Url:      c.AppInfo.EurekaURL + "/eureka/svips/" + svips,
		Method:   http.MethodGet,
		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}

	data := new(Applications)
	err := action.DoRequest(data)
	if err != nil {
		log.Error().AnErr("查询svip下的实例", err)
		return Applications{}, err
	}
	return *data, err
}

func (c *Client) RoundTrip(r *http.Request) (*http.Response, error) {
	hostname := r.URL.Hostname()
	instances := c.serviceCache.Get(hostname)
	if len(instances) == 0 && hostname != "" && net.ParseIP(hostname) == nil {
		app, err := c.GetAppInstances(hostname)
		if err != nil {
			return nil, errors.Wrap(err, "获取服务失败")
		}
		if app.Name != "" {
			for i := range app.Instance {
				c.serviceCache.Set(app.Name, fmt.Sprintf("%s:%d", app.Instance[i].HostName, app.Instance[i].Port.Value))
			}
		}
	}
	go c.refreshAllService()
	return c.originTransport.RoundTrip(r)
}

// 通过request装饰器请求其他服务
func (c *Client) Request(serviceName string, req *http.Request) ([]byte, error) {
	hystrix.DefaultTimeout = 5000
	output := make(chan []byte, 1)
	errors := hystrix.Go(serviceName, func() error {
		return c.callWithRetries(req, output)
	}, func(err error) error {
		if err != nil {
			log.Error().Msgf("回调服务 %s [失败] : %s", serviceName, err)
		}
		return err
	})

	select {
	case out := <-output:
		log.Debug().Msgf("调用服务 %s [成功]", serviceName)
		return out, nil

	case err := <-errors:
		log.Error().Msgf("从通道中得到错误 服务名 %s : %s", serviceName, err)
		return nil, err
	}
}

func (c *Client) startHeartBeat() {

}

var RETRIES = 3

func (c *Client) callWithRetries(req *http.Request, output chan []byte) error {
	r := retrier.New(retrier.ConstantBackoff(RETRIES, 100*time.Millisecond), nil)
	client := &http.Client{Transport: c}
	attempt := 0
	err := r.Run(func() error {
		attempt++
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 299 {
			responseBody, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				output <- responseBody
				return nil
			}
			return err
		} else if err == nil {
			err = fmt.Errorf("不是期待的状态码  [resp.StatusCode=%v]", resp.StatusCode)
		}

		log.Error().Msgf("请求url=%s , 重试失败, 尝试 %v", req.RequestURI, attempt)

		return err
	})
	return err
}

// 公共请求数据
type httpAction struct {
	Method      string
	Url         string
	Body        string
	Accept      string
	ContentType string
	StoreCookie string
	UserName    string
	Password    string
	reader      io.Reader
	HandlerFunc func() ([]byte, error)
}

// 获取请求body值
func (h *httpAction) getRequestBody() io.Reader {
	if h.Body == "" {
		return nil
	}
	h.reader = strings.NewReader(h.Body)
	return h.reader
}

// 发送请求
func (h *httpAction) DoRequest(v interface{}) error {
	var err error
	var body = h.getRequestBody()
	req, err := http.NewRequest(h.Method, h.Url, body)
	if err != nil {
		return errors.Wrap(err, "新建http request失败")
	}
	// Add headers
	req.Header.Add("Accept", h.Accept)
	req.Header.Add("Content-Type", h.ContentType)
	// basic auth
	if h.UserName != "" && h.Password != "" {
		req.SetBasicAuth(h.UserName, h.Password)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Msgf("请求失败 method=%s url=%s err=%v ", req.Method, req.RequestURI, err)
		return errors.Wrap(err, "请求失败")
	}
	defer resp.Body.Close() //nolint
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bytes, _ := ioutil.ReadAll(resp.Body)
		log.Error().Msgf("请求失败: statusCode=%d,method=%s,url=%s,data=%s", resp.StatusCode, req.Method, req.RequestURI, bytes)
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("认证失败: %d, 错误的用户名或密码 ", resp.StatusCode)
		}
		return fmt.Errorf("不是期待的状态码: %d %s", resp.StatusCode, string(bytes))
	}
	if v == nil {
		return err
	}
	caye := resp.Header.Get("Content-Type")
	if "application/xml" == caye {
		err := xml.NewDecoder(resp.Body).Decode(v)
		if err != nil {
			log.Error().Msgf("解析xml数据失败 err=%s", err)
			return fmt.Errorf("解析xml数据失败[%s]", err.Error())
		}
		return err
	}
	if "application/json" == caye {
		return xml.NewDecoder(resp.Body).Decode(v)
	}
	return err
}

// DoWithRetry runs the specified action. If it returns a value, return that value. If it returns an error, sleep for
// sleepBetweenRetries and try again, up to a maximum of maxRetries retries. If maxRetries is exceeded, return a
// MaxRetriesExceeded error.
func doWithRetry(actionDescription string, maxRetries int, sleepBetweenRetries time.Duration, action func() error) error {
	for i := 0; i <= maxRetries; i++ {
		log.Debug().Msg(actionDescription)
		err := action()
		if err == nil {
			return nil
		}

		log.Error().Msgf("%s返回错误: err=%s ,睡眠%s并重试", actionDescription, err.Error(), sleepBetweenRetries)
		time.Sleep(sleepBetweenRetries)
	}

	return MaxRetriesExceeded{Description: actionDescription, MaxRetries: maxRetries}
}

// MaxRetriesExceeded is an error that occurs when the maximum amount of retries is exceeded.
type MaxRetriesExceeded struct {
	Description string
	MaxRetries  int
}

func (err MaxRetriesExceeded) Error() string {
	return fmt.Sprintf("'%s'操作没有成功,将重试%d次", err.Description, err.MaxRetries)
}

func GetLocalIPS() (ips []string) {
	address, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, address := range address {
		if inet, ok := address.(*net.IPNet); ok && !inet.IP.IsLoopback() && inet.IP.To4() != nil {
			ips = append(ips, inet.IP.To4().String())
		}
	}
	return
}

func NowStr() string {
	return time.Now().Format("2006-01-02T15:04:05.000")
}
