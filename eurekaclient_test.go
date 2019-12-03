package goeureka_test

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gorpher/goeureka"
	"github.com/julienschmidt/httprouter"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// eureka 返回的xml数据
var xmldata = `<applications> 
<versions__delta>1</versions__delta>
    <apps__hashcode>UP_2_</apps__hashcode>
    <application>
        <name>SPRING-CLOUD-EUREKA-CLIENT</name>
        <Instance>
            <instanceId>MAX.lan:spring-cloud-eureka-client:8791</instanceId>
            <hostName>192.168.199.239</hostName>
            <app>SPRING-CLOUD-EUREKA-CLIENT</app>
            <ipAddr>192.168.199.239</ipAddr>
            <status>DOWN</status>
            <overriddenstatus>UNKNOWN</overriddenstatus>
            <port enabled="true">8791</port>
            <securePort enabled="false">443</securePort>
            <countryId>1</countryId>
            <dataCenterInfo class="com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo">
                <name>MyOwn</name>
            </dataCenterInfo>
            <leaseInfo>
                <renewalIntervalInSecs>30</renewalIntervalInSecs>
                <durationInSecs>90</durationInSecs>
                <registrationTimestamp>1570688725492</registrationTimestamp>
                <lastRenewalTimestamp>1570688890939</lastRenewalTimestamp>
                <evictionTimestamp>0</evictionTimestamp>
                <serviceUpTimestamp>0</serviceUpTimestamp>
            </leaseInfo>
            <metadata class="java.util.Collections$EmptyMap"/>
            <homePageUrl>http://192.168.199.239:8791/</homePageUrl>
            <statusPageUrl>http://192.168.199.239:8791/info</statusPageUrl>
            <healthCheckUrl>http://192.168.199.239:8791/health</healthCheckUrl>
            <vipAddress>spring-cloud-eureka-client</vipAddress>
            <secureVipAddress>spring-cloud-eureka-client</secureVipAddress>
            <isCoordinatingDiscoveryServer>false</isCoordinatingDiscoveryServer>
            <lastUpdatedTimestamp>1570688725492</lastUpdatedTimestamp>
            <lastDirtyTimestamp>1570688725445</lastDirtyTimestamp>
            <actionType>ADDED</actionType>
        </Instance>
        <Instance>
            <instanceId>MAX.lan:spring-cloud-eureka-client:1000</instanceId>
            <hostName>192.168.199.239</hostName>
            <app>SPRING-CLOUD-EUREKA-CLIENT</app>
            <ipAddr>192.168.199.239</ipAddr>
            <status>UP</status>
            <overriddenstatus>UNKNOWN</overriddenstatus>
            <port enabled="true">1000</port>
            <securePort enabled="false">443</securePort>
            <countryId>1</countryId>
            <dataCenterInfo class="com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo">
                <name>MyOwn</name>
            </dataCenterInfo>
            <leaseInfo>
                <renewalIntervalInSecs>30</renewalIntervalInSecs>
                <durationInSecs>90</durationInSecs>
                <registrationTimestamp>1570688787113</registrationTimestamp>
                <lastRenewalTimestamp>1570688787113</lastRenewalTimestamp>
                <evictionTimestamp>0</evictionTimestamp>
                <serviceUpTimestamp>1570688787113</serviceUpTimestamp>
            </leaseInfo>
            <metadata class="java.util.Collections$EmptyMap"/>
            <homePageUrl>http://192.168.199.239:1000/</homePageUrl>
            <statusPageUrl>http://192.168.199.239:1000/info</statusPageUrl>
            <healthCheckUrl>http://192.168.199.239:1000/health</healthCheckUrl>
            <vipAddress>spring-cloud-eureka-client</vipAddress>
            <secureVipAddress>spring-cloud-eureka-client</secureVipAddress>
            <isCoordinatingDiscoveryServer>false</isCoordinatingDiscoveryServer>
            <lastUpdatedTimestamp>1570688787113</lastUpdatedTimestamp>
            <lastDirtyTimestamp>1570688787035</lastDirtyTimestamp>
            <actionType>ADDED</actionType>
        </Instance>
    </application>
    <application>
        <name>TF-SERVICE1</name>
        <Instance>
            <instanceId>tf-service1:3125</instanceId>
            <hostName>127.0.0.1</hostName>
            <app>TF-SERVICE1</app>
            <ipAddr>127.0.0.1</ipAddr>
            <status>UP</status>
            <overriddenstatus>UNKNOWN</overriddenstatus>
            <port enabled="true">11102</port>
            <securePort enabled="false">7002</securePort>
            <countryId>1</countryId>
            <dataCenterInfo class="com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo">
                <name>MyOwn</name>
            </dataCenterInfo>
            <leaseInfo>
                <renewalIntervalInSecs>30</renewalIntervalInSecs>
                <durationInSecs>90</durationInSecs>
                <registrationTimestamp>1570670943008</registrationTimestamp>
                <lastRenewalTimestamp>1570670943015</lastRenewalTimestamp>
                <evictionTimestamp>0</evictionTimestamp>
                <serviceUpTimestamp>1570670650761</serviceUpTimestamp>
            </leaseInfo>
            <metadata class="java.util.Collections$EmptyMap"/>
            <homePageUrl>http://127.0.0.1:11102/</homePageUrl>
            <statusPageUrl>http://127.0.0.1:11102/info</statusPageUrl>
            <healthCheckUrl>http://127.0.0.1:11102/health</healthCheckUrl>
             <vipAddress>tf-service</vipAddress>
            <secureVipAddress>tf-service</secureVipAddress>
            <isCoordinatingDiscoveryServer>false</isCoordinatingDiscoveryServer>
            <lastUpdatedTimestamp>1570670943008</lastUpdatedTimestamp>
            <lastDirtyTimestamp>1529391461000</lastDirtyTimestamp>
            <actionType>ADDED</actionType>
        </Instance>
    </application>
</applications>`

// 测试解析无数据的xml
func TestNoneParseXmlData(t *testing.T) {
	app := goeureka.Applications{}
	xml.Unmarshal([]byte(""), &app)
	Convey("Check applications data is none", t, func() {
		So(app.AppsHashCode, ShouldBeBlank)
		So(app.VersionDelta, ShouldBeBlank)
		So(app.Application, ShouldBeEmpty)
	})
}

// 测试解析有数据的xml
func TestHaveParseXmlData(t *testing.T) {
	app := new(goeureka.Applications)
	xml.Unmarshal([]byte(xmldata), &app)
	// 第一次转换前检查一次
	checkAppData(t, *app)
	data, err := xml.Marshal(*app)
	// 反转一次后检查
	Convey("check object2xml", t, func() {
		So(err, ShouldBeNil)
		So(data, ShouldNotBeEmpty)
		Convey("check xml2object", func() {
			app = new(goeureka.Applications)
			xml.Unmarshal(data, app)
		})
	})
	checkAppData(t, *app)
}

// 公共验证数据函数
func checkAppData(t *testing.T, app goeureka.Applications) {
	Convey("Check applications data", t, func() {
		So(app.AppsHashCode, ShouldEqual, "UP_2_")
		So(app.VersionDelta, ShouldEqual, "1")
		Convey("Check application data", func() {
			So(app.Application, ShouldHaveLength, 2)
			Convey("Check application[0] data", func() {
				So(app.Application[0], ShouldNotBeEmpty)
				So(app.Application[0].Name, ShouldEqual, "SPRING-CLOUD-EUREKA-CLIENT")
				Convey("Check application[0] Instance data", func() {
					So(app.Application[0].Instance, ShouldNotBeEmpty)
					So(app.Application[0].Instance, ShouldHaveLength, 2)
					Convey("Check application[0] Instance[0] data", func() {

						So(app.Application[0].Instance[0].InstanceID, ShouldEqual, "MAX.lan:spring-cloud-eureka-client:8791")
						So(app.Application[0].Instance[0].HostName, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[0].App, ShouldEqual, "SPRING-CLOUD-EUREKA-CLIENT")
						So(app.Application[0].Instance[0].IpAddr, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[0].Status, ShouldEqual, "DOWN")
						So(app.Application[0].Instance[0].Overriddenstatus, ShouldEqual, "UNKNOWN")
						Convey("Check application[0] Instance port data", func() {
							So(app.Application[0].Instance[0].Port, ShouldNotBeEmpty)
							So(app.Application[0].Instance[0].Port.Enabled, ShouldEqual, true)
							So(app.Application[0].Instance[0].Port.Value, ShouldEqual, 8791)
						})
						Convey("Check application[0] Instance securePort data", func() {
							So(app.Application[0].Instance[0].SecurePort, ShouldNotBeEmpty)
							So(app.Application[0].Instance[0].SecurePort.Enabled, ShouldEqual, false)
							So(app.Application[0].Instance[0].SecurePort.Value, ShouldEqual, 443)
						})

						So(app.Application[0].Instance[0].CountryId, ShouldEqual, 1)
						Convey("Check application[0] Instance dataCenterInfo data", func() {
							So(app.Application[0].Instance[0].DataCenterInfo, ShouldNotBeEmpty)
							So(app.Application[0].Instance[0].DataCenterInfo.Name, ShouldEqual, goeureka.MyOwn)
							So(app.Application[0].Instance[0].DataCenterInfo.Class, ShouldEqual, "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo")
						})
						Convey("Check application[0] Instance LeaseInfo data", func() {
							So(app.Application[0].Instance[0].LeaseInfo, ShouldNotBeEmpty)
							So(app.Application[0].Instance[0].LeaseInfo.RenewalIntervalInSecs, ShouldEqual, 30)
							So(app.Application[0].Instance[0].LeaseInfo.DurationInSecs, ShouldEqual, 90)
							So(app.Application[0].Instance[0].LeaseInfo.RegistrationTimestamp, ShouldEqual, 1570688725492)
							So(app.Application[0].Instance[0].LeaseInfo.LastRenewalTimestamp, ShouldEqual, 1570688890939)
							So(app.Application[0].Instance[0].LeaseInfo.EvictionTimestamp, ShouldEqual, 0)
							So(app.Application[0].Instance[0].LeaseInfo.ServiceUpTimestamp, ShouldEqual, 0)
						})

						So(app.Application[0].Instance[0].Metadata, ShouldBeEmpty)
						So(app.Application[0].Instance[0].HomePageUrl, ShouldEqual, "http://192.168.199.239:8791/")
						So(app.Application[0].Instance[0].StatusPageUrl, ShouldEqual, "http://192.168.199.239:8791/info")
						So(app.Application[0].Instance[0].HealthCheckUrl, ShouldEqual, "http://192.168.199.239:8791/health")
						So(app.Application[0].Instance[0].VipAddress, ShouldEqual, "spring-cloud-eureka-client")
						So(app.Application[0].Instance[0].SecureVipAddress, ShouldEqual, "spring-cloud-eureka-client")
						So(app.Application[0].Instance[0].IsCoordinatingDiscoveryServer, ShouldEqual, false)
						So(app.Application[0].Instance[0].LastUpdatedTimestamp, ShouldEqual, 1570688725492)
						So(app.Application[0].Instance[0].LastDirtyTimestamp, ShouldEqual, 1570688725445)
						So(app.Application[0].Instance[0].ActionType, ShouldEqual, goeureka.ADDED)
					})

					Convey("Check application[0] Instance[1] data", func() {
						So(app.Application[0].Instance[1].InstanceID, ShouldEqual, "MAX.lan:spring-cloud-eureka-client:1000")
						So(app.Application[0].Instance[1].HostName, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[1].App, ShouldEqual, "SPRING-CLOUD-EUREKA-CLIENT")
						So(app.Application[0].Instance[1].IpAddr, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[1].Status, ShouldEqual, "UP")
						So(app.Application[0].Instance[1].Overriddenstatus, ShouldEqual, "UNKNOWN")
						Convey("Check application[0] Instance port data", func() {
							So(app.Application[0].Instance[1].Port, ShouldNotBeEmpty)
							So(app.Application[0].Instance[1].Port.Enabled, ShouldEqual, true)
							So(app.Application[0].Instance[1].Port.Value, ShouldEqual, 1000)
						})
						Convey("Check application[0] Instance securePort data", func() {
							So(app.Application[0].Instance[1].SecurePort, ShouldNotBeEmpty)
							So(app.Application[0].Instance[1].SecurePort.Enabled, ShouldEqual, false)
							So(app.Application[0].Instance[1].SecurePort.Value, ShouldEqual, 443)
						})

						So(app.Application[0].Instance[1].CountryId, ShouldEqual, 1)
						Convey("Check application[0] Instance dataCenterInfo data", func() {
							So(app.Application[0].Instance[1].DataCenterInfo, ShouldNotBeEmpty)
							So(app.Application[0].Instance[1].DataCenterInfo.Name, ShouldEqual, goeureka.MyOwn)
							So(app.Application[0].Instance[1].DataCenterInfo.Class, ShouldEqual, "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo")
						})
						Convey("Check application[0] Instance LeaseInfo data", func() {
							So(app.Application[0].Instance[1].LeaseInfo, ShouldNotBeEmpty)
							So(app.Application[0].Instance[1].LeaseInfo.RenewalIntervalInSecs, ShouldEqual, 30)
							So(app.Application[0].Instance[1].LeaseInfo.DurationInSecs, ShouldEqual, 90)
							So(app.Application[0].Instance[1].LeaseInfo.RegistrationTimestamp, ShouldEqual, 1570688787113)
							So(app.Application[0].Instance[1].LeaseInfo.LastRenewalTimestamp, ShouldEqual, 1570688787113)
							So(app.Application[0].Instance[1].LeaseInfo.EvictionTimestamp, ShouldEqual, 0)
							So(app.Application[0].Instance[1].LeaseInfo.ServiceUpTimestamp, ShouldEqual, 1570688787113)
						})

						So(app.Application[0].Instance[1].Metadata, ShouldBeEmpty)
						So(app.Application[0].Instance[1].HomePageUrl, ShouldEqual, "http://192.168.199.239:1000/")
						So(app.Application[0].Instance[1].StatusPageUrl, ShouldEqual, "http://192.168.199.239:1000/info")
						So(app.Application[0].Instance[1].HealthCheckUrl, ShouldEqual, "http://192.168.199.239:1000/health")
						So(app.Application[0].Instance[1].VipAddress, ShouldEqual, "spring-cloud-eureka-client")
						So(app.Application[0].Instance[1].SecureVipAddress, ShouldEqual, "spring-cloud-eureka-client")
						So(app.Application[0].Instance[1].IsCoordinatingDiscoveryServer, ShouldEqual, false)
						So(app.Application[0].Instance[1].LastUpdatedTimestamp, ShouldEqual, 1570688787113)
						So(app.Application[0].Instance[1].LastDirtyTimestamp, ShouldEqual, 1570688787035)
						So(app.Application[0].Instance[1].ActionType, ShouldEqual, goeureka.ADDED)
					})
				})
			})
			Convey("Check application[1] data", func() {
				So(app.Application[1], ShouldNotBeEmpty)
				So(app.Application[1].Name, ShouldEqual, "TF-SERVICE1")
				Convey("Check application[0] Instance data", func() {
					So(app.Application[1].Instance, ShouldNotBeEmpty)
					So(app.Application[1].Instance, ShouldHaveLength, 1)
					So(app.Application[1].Instance[0].InstanceID, ShouldEqual, "tf-service1:3125")
					So(app.Application[1].Instance[0].HostName, ShouldEqual, "127.0.0.1")
					So(app.Application[1].Instance[0].App, ShouldEqual, "TF-SERVICE1")
					So(app.Application[1].Instance[0].IpAddr, ShouldEqual, "127.0.0.1")
					So(app.Application[1].Instance[0].Status, ShouldEqual, "UP")
					So(app.Application[1].Instance[0].Overriddenstatus, ShouldEqual, "UNKNOWN")
					Convey("Check application[1] Instance port data", func() {
						So(app.Application[1].Instance[0].Port, ShouldNotBeEmpty)
						So(app.Application[1].Instance[0].Port.Enabled, ShouldEqual, true)
						So(app.Application[1].Instance[0].Port.Value, ShouldEqual, 11102)
					})
					Convey("Check application[1] Instance securePort data", func() {
						So(app.Application[1].Instance[0].SecurePort, ShouldNotBeEmpty)
						So(app.Application[1].Instance[0].SecurePort.Enabled, ShouldEqual, false)
						So(app.Application[1].Instance[0].SecurePort.Value, ShouldEqual, 7002)
					})

					So(app.Application[1].Instance[0].CountryId, ShouldEqual, 1)
					Convey("Check application[1] Instance dataCenterInfo data", func() {
						So(app.Application[1].Instance[0].DataCenterInfo, ShouldNotBeEmpty)
						So(app.Application[1].Instance[0].DataCenterInfo.Name, ShouldEqual, goeureka.MyOwn)
						So(app.Application[1].Instance[0].DataCenterInfo.Class, ShouldEqual, "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo")
					})
					Convey("Check application[1] Instance LeaseInfo data", func() {
						So(app.Application[1].Instance[0].LeaseInfo, ShouldNotBeEmpty)
						So(app.Application[1].Instance[0].LeaseInfo.RenewalIntervalInSecs, ShouldEqual, 30)
						So(app.Application[1].Instance[0].LeaseInfo.DurationInSecs, ShouldEqual, 90)
						So(app.Application[1].Instance[0].LeaseInfo.RegistrationTimestamp, ShouldEqual, 1570670943008)
						So(app.Application[1].Instance[0].LeaseInfo.LastRenewalTimestamp, ShouldEqual, 1570670943015)
						So(app.Application[1].Instance[0].LeaseInfo.EvictionTimestamp, ShouldEqual, 0)
						So(app.Application[1].Instance[0].LeaseInfo.ServiceUpTimestamp, ShouldEqual, 1570670650761)
					})

					So(app.Application[1].Instance[0].Metadata, ShouldBeEmpty)
					So(app.Application[1].Instance[0].HomePageUrl, ShouldEqual, "http://127.0.0.1:11102/")
					So(app.Application[1].Instance[0].StatusPageUrl, ShouldEqual, "http://127.0.0.1:11102/info")
					So(app.Application[1].Instance[0].HealthCheckUrl, ShouldEqual, "http://127.0.0.1:11102/health")
					So(app.Application[1].Instance[0].VipAddress, ShouldEqual, "tf-service")
					So(app.Application[1].Instance[0].SecureVipAddress, ShouldEqual, "tf-service")
					So(app.Application[1].Instance[0].IsCoordinatingDiscoveryServer, ShouldEqual, false)
					So(app.Application[1].Instance[0].LastUpdatedTimestamp, ShouldEqual, 1570670943008)
					So(app.Application[1].Instance[0].LastDirtyTimestamp, ShouldEqual, 1529391461000)
					So(app.Application[1].Instance[0].ActionType, ShouldEqual, goeureka.ADDED)
				})
			})
		})
	})
}

type mockDB goeureka.Applications

// 注册实例
func (m *mockDB) Register(instance goeureka.Instance) {
	if m.Application == nil || len(m.Application) == 0 {
		apps := make([]goeureka.Application, 1)
		apps[0] = goeureka.Application{Name: instance.App, Instance: []goeureka.Instance{
			instance,
		}}
		m.Application = apps
		return
	}

	appUpdate := false
	for i := range m.Application {
		//如果是同一个应用
		if m.Application[i].Name == instance.App {
			instanceUpdate := false
			for k := range m.Application[i].Instance {
				//如果是同一个实例,进行更新
				if m.Application[i].Instance[k].InstanceID == instance.InstanceID {
					m.Application[i].Instance[k] = instance
					instanceUpdate = true
					break
				}
			}
			// 添加实例
			if !instanceUpdate {
				m.Application[i].Instance = append(m.Application[i].Instance, instance)
			}
			appUpdate = true
			break
		}
	}

	// 没有更新应用说明是新添加的应用
	if !appUpdate {
		app := goeureka.Application{Name: instance.App, Instance: []goeureka.Instance{
			instance,
		}}
		m.Application = append(m.Application, app)
	}
}

// 获取所有实例
func (m *mockDB) GetApps() goeureka.Applications {
	if len(m.Application) > 0 {
		m.AppsHashCode = string(goeureka.UP) + "_" + strconv.Itoa(len(m.Application)) + "_"
	}
	return goeureka.Applications(*m)
}

func (m *mockDB) Delete(appId string, instanceId string) bool {
	if len(m.Application) == 0 {
		return false
	}
	for i := range m.Application {
		if m.Application[i].Name == appId {
			for a := range m.Application[i].Instance {
				if m.Application[i].Instance[a].InstanceID == instanceId {
					if len(m.Application[i].Instance) == 1 {
						m.Application = m.Application[:0]
						return true
					}
					m.Application[i].Instance = append(m.Application[i].Instance[:a], m.Application[i].Instance[a+1:]...)
					return true
				}
			}
			return true
		}
	}
	return false
}

func (m *mockDB) GetAppByID(id string) (goeureka.Application, bool) {
	if len(m.Application) == 0 {
		return goeureka.Application{}, false
	}
	for i := range m.Application {
		if m.Application[i].Name == id {
			return m.Application[i], true
		}
	}
	return goeureka.Application{}, false
}

func (m *mockDB) InstanceExist(appID string, instanceID string) bool {
	for a := range m.Application {
		if m.Application[a].Name == appID {
			for i := range m.Application[a].Instance {
				if m.Application[a].Instance[i].InstanceID == instanceID {
					return true
				}
			}
		}
	}
	return false
}

func (m *mockDB) AppExist(appID string) bool {
	for a := range m.Application {
		if m.Application[a].Name == appID {
			return true
		}
	}
	return false
}

func (m *mockDB) GetAppInstanceByID(appID string, instanceID string) goeureka.Instance {
	for i := range m.Application {
		if m.Application[i].Name == appID {
			for k := range m.Application[i].Instance {
				if m.Application[i].Instance[k].InstanceID == instanceID {
					return m.Application[i].Instance[k]
				}
			}
		}
	}
	return goeureka.Instance{}
}

func (m *mockDB) UpdateInstanceStatus(appID string, instanceID string, status string) {
	for k := range m.Application {
		if m.Application[k].Name == appID {
			for i := range m.Application[k].Instance {
				if m.Application[k].Instance[i].InstanceID == instanceID {
					m.Application[k].Instance[i].Status = goeureka.InstanceStatus(status)
				}
			}
		}
	}
}

func (m *mockDB) UpdateInstanceMetadata(appID string, instanceID string, metadata map[string]string) {
	for k := range m.Application {
		if m.Application[k].Name == appID {
			for i := range m.Application[k].Instance {
				if m.Application[k].Instance[i].InstanceID == instanceID {
					m.Application[k].Instance[i].Metadata = metadata
				}
			}
		}
	}
}

func (m *mockDB) getAppsByInstanceFilter(filterInstance func(goeureka.Instance) bool) goeureka.Applications {
	apps := goeureka.Applications{
		AppsHashCode: "",
		VersionDelta: "-1",
	}
	for k := range m.Application {
		for i := range m.Application[k].Instance {

			if filterInstance(m.Application[k].Instance[i]) {
				newApps := []goeureka.Application{{
					Name:     m.Application[k].Name,
					Instance: []goeureka.Instance{m.Application[k].Instance[i]}}}
				if apps.Application != nil && len(apps.Application) == 0 {
					apps.Application = newApps
					continue
				}
				for a := range apps.Application {
					if apps.Application[a].Name == m.Application[k].Name && len(apps.Application[a].Instance) > 0 {
						apps.Application[a].Instance = append(apps.Application[a].Instance, newApps[0].Instance[0])
					}
					continue
				}
				apps.Application = append(apps.Application, newApps[0])
			}
		}
	}
	if len(apps.Application) > 0 {
		apps.AppsHashCode = string(goeureka.UP) + "_" + strconv.Itoa(len(apps.Application)) + "_"
	}
	return apps
}
func (m *mockDB) GetVips(vips string) goeureka.Applications {
	return m.getAppsByInstanceFilter(func(instance goeureka.Instance) bool {
		if instance.VipAddress == vips {
			return true
		}
		return false
	})
}

func (m *mockDB) GetSVips(svips string) goeureka.Applications {
	return m.getAppsByInstanceFilter(func(instance goeureka.Instance) bool {
		if instance.SecureVipAddress == svips {
			return true
		}
		return false
	})
}

func MockRouter() *httprouter.Router {
	type RequestInstanceModel struct {
		Instance goeureka.Instance `json:"instance"`
	}
	router := httprouter.New()

	var MockDB = mockDB{
		VersionDelta: "1",
	}

	// 测试路由
	router.Handle(http.MethodGet, "/hello/:ID", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			fmt.Fprint(w, p.ByName("ID"))
		}
	})

	// 注册上线
	//POST {{url}}/eureka/apps/{{appID}}
	router.Handle(http.MethodPost, "/eureka/apps/:appID", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {

			ins := new(RequestInstanceModel)
			err := json.NewDecoder(r.Body).Decode(ins)
			if err != nil {
				w.WriteHeader(http.StatusPaymentRequired)
				w.Write([]byte(err.Error()))
				return
			}
			MockDB.Register(ins.Instance)
		}
	})

	// 发送心跳
	// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}
	router.Handle(http.MethodPut, "/eureka/apps/:appID/:instanceID", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			if !MockDB.InstanceExist(p.ByName("appID"), p.ByName("instanceID")) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			data := new(RequestInstanceModel)
			err := json.NewDecoder(r.Body).Decode(data)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			MockDB.Register(data.Instance)
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	// 获取所有实例
	// GET {{url}}/eureka/apps
	router.Handle(http.MethodGet, "/eureka/apps", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			w.Header().Set("Content-Type", "application/xml")
			apps := MockDB.GetApps()
			date, err := xml.Marshal(apps)
			if err != nil {
				w.WriteHeader(http.StatusPaymentRequired)
				w.Write([]byte(err.Error()))
				return
			}
			w.Write(date)
		}
	})

	// 下线
	// DELETE {{url}}/eureka/apps/{{appID}}/{{instanceID}}
	router.Handle(http.MethodDelete, "/eureka/apps/:appID/:instanceID", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			if MockDB.Delete(p.ByName("appID"), p.ByName("instanceID")) {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})

	// 查询服务的所有实例
	// GET {{url}}/eureka/apps/{{appID}}
	router.Handle(http.MethodGet, "/eureka/apps/:appID", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			app, flag := MockDB.GetAppByID(p.ByName("appID"))
			if flag {
				w.Header().Set("Content-Type", "application/xml")

				err := xml.NewEncoder(w).Encode(app)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// 查询指定服务实例
	// GET {{url}}/eureka/apps/{{appID}}/{{instanceID}}
	router.Handle(http.MethodGet, "/eureka/apps/:appID/:instanceID", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			if !MockDB.InstanceExist(p.ByName("appID"), p.ByName("instanceID")) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			instance := MockDB.GetAppInstanceByID(p.ByName("appID"), p.ByName("instanceID"))
			w.Header().Set("Content-Type", "application/xml")
			err := xml.NewEncoder(w).Encode(instance)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return
		}
	})

	//更改实例状态,移除实例
	// PUT {{url}}/eureka/apps/{{appID}}/{{instanceID}}/status?value=OUT_OF_SERVICE
	router.Handle(http.MethodPut, "/eureka/apps/:appID/:instanceID/status", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			if !MockDB.InstanceExist(p.ByName("appID"), p.ByName("instanceID")) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			status := r.URL.Query().Get("value")
			MockDB.UpdateInstanceStatus(p.ByName("appID"), p.ByName("instanceID"), status)
			w.WriteHeader(http.StatusOK)
			return
		}
	})
	//更新元数据
	//{{url}}/eureka/v2/apps/{{appID}}/{{instanceID}}/metadata?key=value
	router.Handle(http.MethodPut, "/eureka/apps/:appID/:instanceID/metadata", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		log.Println(p.ByName("appID"))
		if auth(r, w) {
			if !MockDB.InstanceExist(p.ByName("appID"), p.ByName("instanceID")) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			values := r.URL.Query()
			m := make(map[string]string, len(values))
			for k := range values {
				m[k] = values[k][0]
			}
			MockDB.UpdateInstanceMetadata(p.ByName("appID"), p.ByName("instanceID"), m)
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	// 查询vip下的实例
	//GET {{url}}/eureka/vips/{{vipAddress}}
	router.Handle(http.MethodGet, "/eureka/vips/:vipAddress", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			apps := MockDB.GetVips(p.ByName("vipAddress"))
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			err := xml.NewEncoder(w).Encode(apps)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	})

	// 查询svip下的实例
	//GET {{url}}/eureka/svips/{{svipAddress}}
	router.Handle(http.MethodGet, "/eureka/svips/:svipAddress", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if auth(r, w) {
			apps := MockDB.GetSVips(p.ByName("svipAddress"))
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			err := xml.NewEncoder(w).Encode(apps)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	})
	return router
}

// 用户认证
// 用户名 密码 user pass123
func auth(r *http.Request, w http.ResponseWriter) bool {
	if g, e := r.Header.Get("Authorization"), "Basic dXNlcjpwYXNzMTIz"; g != e {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, fmt.Sprintf("got header %q, want %q", g, e))
		return false
	}
	return true
}

// 测试 xml格式数据转对象再转json测试
func TestParseXml2Json(t *testing.T) {
	app := new(goeureka.Applications)
	app2 := new(goeureka.Applications)
	xml.Unmarshal([]byte(xmldata), app)
	data, err := json.Marshal(*app)
	Convey("check object2json", t, func() {
		So(err, ShouldBeNil)
		So(data, ShouldNotBeEmpty)
		Convey("check json2object", func() {
			json.Unmarshal(data, &app2)
		})
	})
	checkAppData(t, *app2)
}

// 测试路由
func TestRouter(t *testing.T) {
	router := MockRouter()
	ts := httptest.NewServer(router)
	defer ts.Close()
	appID := "SPRING-CLOUD-EUREKA-CLIENT"
	r, _ := http.NewRequest(http.MethodGet, ts.URL+"/hello/"+appID, nil)
	r.SetBasicAuth("user", "pass123")
	resp, err := http.DefaultClient.Do(r)
	//resp, err := ts.Client().Do(r) //使用client也可以
	require.Nil(t, err)
	bytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	require.Nil(t, err)
	require.Contains(t, string(bytes), appID)
}

// 获取测试GoEurekaClient
func testNewGoEurekaClient(t *testing.T) (*goeureka.Client, func()) {
	router := MockRouter()
	ts := httptest.NewServer(router)
	url, err := url.Parse(ts.URL)
	require.Nil(t, err)
	port, err := strconv.Atoi(url.Port())
	require.Nil(t, err)
	client, err := goeureka.New(&goeureka.AppInfo{
		AppID:     "SPRING-CLOUD-EUREKA-CLIENT",
		HostName:  url.Hostname(),
		Port:      port,
		UserName:  "user",
		Password:  "pass123",
		EurekaURL: ts.URL,
	})
	require.Nil(t, err)
	return client, func() {
		ts.Close()
	}
}

//注册实例
func TestClient_Register(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
}

// 获取所有实例
func TestClient_GetApps(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
	apps, err := client.GetApps()
	require.Nil(t, err)
	require.NotNil(t, apps)
	require.Equal(t, len(apps.Application), 1)
}

// 下线
func TestClient_Deregister(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
	apps, err := client.GetApps()
	require.Nil(t, err)
	require.NotNil(t, apps)
	require.Equal(t, len(apps.Application), 1)
	err = client.Deregister()
	require.Nil(t, err)
	apps, err = client.GetApps()
	require.Nil(t, err)
	require.NotNil(t, apps)
	require.Equal(t, len(apps.Application), 0)
}

//获取所有实例
func TestClient_GetAppInstances(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
	apps, err := client.GetOwnAppInstances()
	require.Nil(t, err)
	require.NotNil(t, apps)
	require.Equal(t, len(apps.Instance), 1)
	app2, err := client.GetOwnAppInstance()
	require.Nil(t, err)
	require.Equal(t, apps.Instance[0], app2)
}

// 根据ID获取实例信息
func TestClient_GetAppInstanceByID(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
	apps, err := client.GetAppInstanceByID(client.AppInfo.AppID, client.AppInfo.InstanceID)
	require.Nil(t, err)
	require.NotNil(t, apps)
	app2, err := client.GetOwnAppInstance()
	require.Nil(t, err)
	require.Equal(t, apps, app2)
}

// 更新状态
func TestClient_UpdateOwnAppInstanceStatus(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
	err = client.UpdateOwnAppInstanceStatus(goeureka.DOWN)
	require.Nil(t, err)
	app2, err := client.GetOwnAppInstance()
	require.Nil(t, err)
	require.Equal(t, goeureka.DOWN, app2.Status)
}

// 根据vip获取应用
func TestClient_GetAppInstanceVip(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
	vip, err := client.GetAppInstanceVip("127.0.0.1")
	require.Nil(t, err)
	require.NotNil(t, vip.Application)
	require.Equal(t, 1, len(vip.Application))
}

func TestClient_GetAppInstanceSVip(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()
	err := client.Register()
	require.Nil(t, err)
	_, err = client.GetAppInstanceSVip("127.0.0.1")
	require.Nil(t, err)

}

// 测试没有注册任何服务
func Test404(t *testing.T) {
	client, fn := testNewGoEurekaClient(t)
	defer fn()

	app, err := client.GetApps()
	require.Nil(t, err)
	require.Equal(t, "1", app.VersionDelta)

	err = client.Deregister()
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "404")

	instances, err := client.GetOwnAppInstances()
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "404")
	require.Empty(t, instances.Name)
	require.Nil(t, instances.Instance)

	ins, err := client.GetAppInstanceByID("appId", "instanceId")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "404")
	require.Empty(t, ins.App)

	instances, err = client.GetOwnAppInstances()
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "404")
	require.Empty(t, instances.Name)
	require.Nil(t, instances.Instance)

	err = client.UpdateOwnAppInstanceStatus(goeureka.OUT_OF_SERVICE)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "404")

	m := map[string]string{
		"score": "good",
	}
	err = client.UpdateOwnAppInstanceMetadata(m)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "404")

	apps, err := client.GetAppInstanceVip("127.0.0.1")
	require.Nil(t, err)
	require.Equal(t, "-1", apps.VersionDelta)

	apps, err = client.GetAppInstanceSVip("127.0.0.1")
	require.Nil(t, err)
	require.Equal(t, "-1", apps.VersionDelta)
}
