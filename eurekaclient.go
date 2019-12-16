package goeureka

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
)

type serviceCache struct {
	v map[string]map[int64]string
	sync.RWMutex
	timeout time.Duration
}

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
func (s *serviceCache) Delete(k string) {
	s.Lock()
	defer s.Unlock()
	if len(s.v) > 0 {
		delete(s.v, k)
	}
}

// eureka客户端
type Client struct {
	AppInfo         AppInfo
	Instance        Instance
	serviceCache    *serviceCache
	heartbeats      bool          //心跳中
	stopChan        chan struct{} //停止发送心跳
	originTransport http.RoundTripper
}

var defaultUrl = "http://localhost:8761" //默认注册中心地址

// 新建goeureka客户端
func New(appInfo *AppInfo) (*Client, error) {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	})
	zerolog.SetGlobalLevel(appInfo.LogLevel)
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
		AppInfo: AppInfo{},
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
		serviceCache:    NewServiceCache(time.Second * 5),
		originTransport: http.DefaultTransport,
	}
	c.AppInfo = *appInfo
	hostname, _ := os.Hostname() // nolint
	if c.AppInfo.HostName == "" {
		ips, err := net.LookupIP(hostname)
		if err == nil && len(ips) > 0 {
			for i := range ips {
				if !ips[i].IsLoopback() && ips[i].To4() != nil {
					c.AppInfo.HostName = ips[i].To4().String()
					break
				}
			}
		}
	}
	if hostname == "" {
		hostname = randomStr(12)
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
	c.Instance.IpAddr = c.AppInfo.HostName
	c.Instance.Status = UP
	c.Instance.Overriddenstatus = UP
	c.Instance.Port.Value = c.AppInfo.Port
	c.Instance.SecurePort.Value = 443

	c.Instance.HomePageUrl = "http://" + c.Instance.HostName + ":" + strconv.Itoa(c.Instance.Port.Value) + "/"
	c.Instance.StatusPageUrl = "http://" + c.Instance.HostName + ":" + strconv.Itoa(c.Instance.Port.Value) + "/info"
	c.Instance.HealthCheckUrl = "http://" + c.Instance.HostName + ":" + strconv.Itoa(c.Instance.Port.Value) + "/health"

	c.Instance.VipAddress = c.Instance.HostName
	c.Instance.SecureVipAddress = GetLocalIP()
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
		log.Print(" Instance mode to json err:", err)
		return err
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
		log.Print("Trying to Register to Eureka Server...")
		err = action.DoRequest(nil)
		if err == nil {
			log.Print("Register success...")
			return nil
		}
		return err
	})
	if err != nil {
		log.Print("registry err: ", err)
		return err
	}
	// 刷新获取所有服务加入到缓存
	go c.refreshAllService()

	// 每隔5秒尝试注册到eureka,直到注册成功.
	go c.StartHeartbeat() //开始发送心跳
	return err
}

// 刷新所有服务
func (c *Client) refreshAllService() {
	apps, err := c.GetApps()
	if err != nil {
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

// 停止发送心跳
func (c *Client) StopHeartbeat() {
	c.stopChan <- struct{}{}
}

// 发送心跳
// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}
func (c *Client) StartHeartbeat() error {
	c.heartbeats = true
	inst := struct {
		Instance Instance `json:"instance"`
	}{
		c.Instance,
	}
	data, err := json.Marshal(inst)
	if err != nil {
		log.Print(" Instance mode to json err:", err)
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
		log.Print("Trying to send Instance heart beat...")
		err := action.DoRequest(nil)
		if err != nil {
			log.Print("send Instance heart beat failed!!!")
			return err
		}
		log.Print("send Instance heart beat success!")
		return nil
	}
	for {
		// 每隔30秒发送一次心跳
		err := heartbeat()
		if err != nil {
			log.Print("send heart beat err: ", err)
		}
		select {
		case <-time.After(30 * time.Second):
		case <-c.stopChan:
			c.heartbeats = false
			return nil
		}
	}
}

// 下线
// DELETE {{url}}/eureka/apps/{{appID}}/{{instanceID}}
func (c *Client) Deregister() error {
	log.Print("Trying to deregister application...")
	// 停止发送心跳
	c.stopChan <- struct{}{}
	// Deregister
	action := httpAction{
		Url:         c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App + "/" + c.Instance.InstanceID,
		Method:      http.MethodDelete,
		ContentType: "application/json",
		UserName:    c.AppInfo.UserName,
		Password:    c.AppInfo.Password,
	}
	err := action.DoRequest(nil)
	if err != nil {
		log.Print("Deregistered application, exiting. Check Client...")
		return err
	}
	log.Print("deregister application success")
	return nil
}

// 获取所有实例
// GET {{url}}/eureka/apps
func (c *Client) GetApps() (Applications, error) {
	log.Print("Trying to query for all appID instances...")
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
		log.Print("get applications err")
		return Applications{}, err
	}

	return *data, err
}

// 获取指定服务的实例
// GET {{url}}/eureka/apps/{{appID}}
func (c *Client) GetAppInstances(appID string) (Application, error) {
	log.Print("Trying to query for a specific appID/instanceID...")
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
		return Application{}, err
	}
	return *data, err
}

// 查询同服务的所有实例
// GET {{url}}/eureka/apps/{{appID}}
func (c *Client) GetOwnAppInstances() (Application, error) {
	log.Print("Trying to query for a specific appID/instanceID...")
	action := httpAction{
		Url:      c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App,
		Method:   "GET",
		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}
	data := new(Application)
	err := action.DoRequest(data)
	if err != nil {
		return Application{}, err
	}
	return *data, err
}

// 查询自己注册的服务信息
// GET {{url}}/eureka/apps/{{appID}}/{{appID}}:{{instanceID}}
func (c *Client) GetOwnAppInstance() (Instance, error) {
	log.Print("Trying to request Own application Instance detail...")
	action := httpAction{
		Url:      c.AppInfo.EurekaURL + "/eureka/apps/" + c.Instance.App + "/" + c.Instance.InstanceID,
		Method:   "GET",
		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}
	data := new(Instance)
	err := action.DoRequest(data)
	if err != nil {
		return Instance{}, err
	}
	return *data, err
}

// 查询指定服务实例
func (c *Client) GetAppInstanceByID(appID string, instanceID string) (Instance, error) {
	log.Print("Trying to request Own application Instance detail...")
	action := httpAction{
		Url:      c.AppInfo.EurekaURL + "/eureka/apps/" + appID + "/" + instanceID,
		Method:   http.MethodGet,
		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}
	data := new(Instance)
	err := action.DoRequest(data)
	if err != nil {
		return Instance{}, err
	}
	return *data, err
}

//更改自己实例状态
// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}/status?value=OUT_OF_SERVICE
func (c *Client) UpdateOwnAppInstanceStatus(status InstanceStatus) error {
	log.Print("Trying to change Instance status ...")
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
		log.Print("更改状态下线:", DOWN)
		if c.heartbeats {
			c.stopChan <- struct{}{}
		}
		for c.heartbeats {
		}
	case OUT_OF_SERVICE:
		log.Print("更改状态下线:", OUT_OF_SERVICE)
		if c.heartbeats {
			c.stopChan <- struct{}{}
		}
		for c.heartbeats {
		}
	case UNKNOWN:
	case UP:
		if !c.heartbeats {
			doHeartBeat = true
		}
	case STARTING:
	}
	err := action.DoRequest(nil)
	if err == nil && doHeartBeat {
		log.Print("重新开始发送心跳...")
		c.StartHeartbeat()
	}
	return err
}

//更新元数据
//{{url}}/eureka/v2/apps/{{appID}}/{{instanceID}}/metadata?key=value
func (c *Client) UpdateOwnAppInstanceMetadata(value map[string]string) error {
	log.Print("Trying to update metadata ...")
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
	err := action.DoRequest(nil)
	return err
}

// 查询vip下的实例
//GET {{url}}/eureka/vips/{{vipAddress}}
func (c *Client) GetAppInstanceVip(vip string) (Applications, error) {
	log.Print("Trying to query Instance in the a svip ...")
	action := httpAction{
		Url:      c.AppInfo.EurekaURL + "/eureka/vips/" + vip,
		Method:   http.MethodGet,
		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}
	data := new(Applications)
	err := action.DoRequest(data)
	if err != nil {
		return Applications{}, err
	}
	return *data, err
}

// 查询svip下的实例
//GET {{url}}/eureka/svips/{{svipAddress}}
func (c *Client) GetAppInstanceSVip(svips string) (Applications, error) {
	log.Print("Trying to query Instance in the a svip ...")
	action := httpAction{
		Url:      c.AppInfo.EurekaURL + "/eureka/svips/" + svips,
		Method:   http.MethodGet,
		UserName: c.AppInfo.UserName,
		Password: c.AppInfo.Password,
	}

	data := new(Applications)
	err := action.DoRequest(data)
	if err != nil {
		return Applications{}, err
	}
	return *data, err
}

// 通过自己定义请求其他服务
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	client := &http.Client{
		Transport: c,
		Timeout:   time.Second * 5,
	}
	return client.Do(r)
}

func (c *Client) RoundTrip(r *http.Request) (*http.Response, error) {
	hostname := r.URL.Hostname()
	instances := c.serviceCache.Get(hostname)
	if len(instances) == 0 && hostname != "" {
		app, err := c.GetAppInstances(hostname)
		if err != nil {
			return nil, ErrorNotFoundService.Wrap(err)
		}
		if app.Name != "" {
			for i := range app.Instance {
				c.serviceCache.Set(app.Name, fmt.Sprintf("%s:%d", app.Instance[i].HostName, app.Instance[i].Port.Value))
			}
		}
	}
	homeric := c.serviceCache.Get(hostname)
	if homeric == "" {
		return &http.Response{}, fmt.Errorf("没有找到服务: %s", hostname)
	}
	host := strings.ReplaceAll(r.URL.Host, hostname, homeric)
	r.URL.Host = host
	go c.refreshAllService()
	return c.originTransport.RoundTrip(r)
}

// 通过request装饰器请求其他服务
func (c *Client) Request(req *http.Request) (*http.Response, error) {
	client := &http.Client{Transport: &http.Transport{}}
	return client.Do(req)
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
		return err
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
		bytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("HTTP request failed: %s %s", err, string(bytes))
		return fmt.Errorf("发送请求失败: %s", err.Error())
	}
	defer resp.Body.Close() //nolint
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("HTTP request failed: statusCode=%d ,%s ", resp.StatusCode, string(bytes))
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
			log.Printf("Parser response xml data failed: %s", err)
			return fmt.Errorf("解析xml返回值失败: %s", err.Error())
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
		log.Print(actionDescription)

		err := action()
		if err == nil {
			return nil
		}

		log.Printf("%s returned an error: %s. Sleeping for %s and will try again.\n", actionDescription, err.Error(), sleepBetweenRetries)
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
	return fmt.Sprintf("'%s' unsuccessful after %d retries", err.Description, err.MaxRetries)
}

func GetLocalIP() string {
	adders, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range adders {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
