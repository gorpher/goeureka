package goeureka_test

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wx11055/goeureka"
	"github.com/wx11055/goeureka/model"
	"github.com/wx11055/gone/logger"
	"gopkg.in/h2non/gock.v1"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// eureka 返回的xml数据
var xmldata = `<applications> 
<versions__delta>1</versions__delta>
    <apps__hashcode>UP_2_</apps__hashcode>
    <application>
        <name>SPRING-CLOUD-EUREKA-CLIENT</name>
        <instance>
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
        </instance>
        <instance>
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
        </instance>
    </application>
    <application>
        <name>TF-SERVICE1</name>
        <instance>
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
        </instance>
    </application>
</applications>`

// 测试解析无数据的xml
func TestNoneParseXmlData(t *testing.T) {
	app := model.Applications{}
	xml.Unmarshal([]byte(""), &app)
	Convey("Check applications data is none", t, func() {
		So(app.AppsHashCode, ShouldBeBlank)
		So(app.VersionDelta, ShouldBeBlank)
		So(app.Application, ShouldBeEmpty)
	})
}

// 测试解析有数据的xml
func TestHaveParseXmlData(t *testing.T) {
	app := new(model.Applications)
	xml.Unmarshal([]byte(xmldata), &app)
	// 第一次转换前检查一次
	checkAppData(t, *app)
	//xml.Unmarshal([]byte(xmldata), &app)
	data, err := xml.Marshal(*app)
	// 反转一次后检查
	Convey("check object2xml", t, func() {
		So(err, ShouldBeNil)
		So(data, ShouldNotBeEmpty)
		Convey("check xml2object", func() {
			app = new(model.Applications)
			xml.Unmarshal(data, app)
		})
	})

	checkAppData(t, *app)
}

// 公共验证数据函数
func checkAppData(t *testing.T, app model.Applications) {
	Convey("Check applications data", t, func() {
		So(app.AppsHashCode, ShouldEqual, "UP_2_")
		So(app.VersionDelta, ShouldEqual, "1")
		Convey("Check application data", func() {
			So(app.Application, ShouldHaveLength, 2)
			Convey("Check application[0] data", func() {
				So(app.Application[0], ShouldNotBeEmpty)
				So(app.Application[0].Name, ShouldEqual, "SPRING-CLOUD-EUREKA-CLIENT")
				Convey("Check application[0] instance data", func() {
					So(app.Application[0].Instance, ShouldNotBeEmpty)
					So(app.Application[0].Instance, ShouldHaveLength, 2)
					Convey("Check application[0] instance[0] data", func() {

						So(app.Application[0].Instance[0].InstanceID, ShouldEqual, "MAX.lan:spring-cloud-eureka-client:8791")
						So(app.Application[0].Instance[0].HostName, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[0].App, ShouldEqual, "SPRING-CLOUD-EUREKA-CLIENT")
						So(app.Application[0].Instance[0].IpAddr, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[0].Status, ShouldEqual, "DOWN")
						So(app.Application[0].Instance[0].Overriddenstatus, ShouldEqual, "UNKNOWN")
						Convey("Check application[0] instance port data", func() {
							So(app.Application[0].Instance[0].Port, ShouldNotBeEmpty)
							So(app.Application[0].Instance[0].Port.Enabled, ShouldEqual, true)
							So(app.Application[0].Instance[0].Port.Value, ShouldEqual, 8791)
						})
						Convey("Check application[0] instance securePort data", func() {
							So(app.Application[0].Instance[0].SecurePort, ShouldNotBeEmpty)
							So(app.Application[0].Instance[0].SecurePort.Enabled, ShouldEqual, false)
							So(app.Application[0].Instance[0].SecurePort.Value, ShouldEqual, 443)
						})

						So(app.Application[0].Instance[0].CountryId, ShouldEqual, 1)
						Convey("Check application[0] instance dataCenterInfo data", func() {
							So(app.Application[0].Instance[0].DataCenterInfo, ShouldNotBeEmpty)
							So(app.Application[0].Instance[0].DataCenterInfo.Name, ShouldEqual, model.MyOwn)
							So(app.Application[0].Instance[0].DataCenterInfo.Class, ShouldEqual, "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo")
						})
						Convey("Check application[0] instance LeaseInfo data", func() {
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
						So(app.Application[0].Instance[0].ActionType, ShouldEqual, model.ADDED)
					})

					Convey("Check application[0] instance[1] data", func() {
						So(app.Application[0].Instance[1].InstanceID, ShouldEqual, "MAX.lan:spring-cloud-eureka-client:1000")
						So(app.Application[0].Instance[1].HostName, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[1].App, ShouldEqual, "SPRING-CLOUD-EUREKA-CLIENT")
						So(app.Application[0].Instance[1].IpAddr, ShouldEqual, "192.168.199.239")
						So(app.Application[0].Instance[1].Status, ShouldEqual, "UP")
						So(app.Application[0].Instance[1].Overriddenstatus, ShouldEqual, "UNKNOWN")
						Convey("Check application[0] instance port data", func() {
							So(app.Application[0].Instance[1].Port, ShouldNotBeEmpty)
							So(app.Application[0].Instance[1].Port.Enabled, ShouldEqual, true)
							So(app.Application[0].Instance[1].Port.Value, ShouldEqual, 1000)
						})
						Convey("Check application[0] instance securePort data", func() {
							So(app.Application[0].Instance[1].SecurePort, ShouldNotBeEmpty)
							So(app.Application[0].Instance[1].SecurePort.Enabled, ShouldEqual, false)
							So(app.Application[0].Instance[1].SecurePort.Value, ShouldEqual, 443)
						})

						So(app.Application[0].Instance[1].CountryId, ShouldEqual, 1)
						Convey("Check application[0] instance dataCenterInfo data", func() {
							So(app.Application[0].Instance[1].DataCenterInfo, ShouldNotBeEmpty)
							So(app.Application[0].Instance[1].DataCenterInfo.Name, ShouldEqual, model.MyOwn)
							So(app.Application[0].Instance[1].DataCenterInfo.Class, ShouldEqual, "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo")
						})
						Convey("Check application[0] instance LeaseInfo data", func() {
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
						So(app.Application[0].Instance[1].ActionType, ShouldEqual, model.ADDED)
					})
				})
			})
			Convey("Check application[1] data", func() {
				So(app.Application[1], ShouldNotBeEmpty)
				So(app.Application[1].Name, ShouldEqual, "TF-SERVICE1")
				Convey("Check application[0] instance data", func() {
					So(app.Application[1].Instance, ShouldNotBeEmpty)
					So(app.Application[1].Instance, ShouldHaveLength, 1)
					So(app.Application[1].Instance[0].InstanceID, ShouldEqual, "tf-service1:3125")
					So(app.Application[1].Instance[0].HostName, ShouldEqual, "127.0.0.1")
					So(app.Application[1].Instance[0].App, ShouldEqual, "TF-SERVICE1")
					So(app.Application[1].Instance[0].IpAddr, ShouldEqual, "127.0.0.1")
					So(app.Application[1].Instance[0].Status, ShouldEqual, "UP")
					So(app.Application[1].Instance[0].Overriddenstatus, ShouldEqual, "UNKNOWN")
					Convey("Check application[1] instance port data", func() {
						So(app.Application[1].Instance[0].Port, ShouldNotBeEmpty)
						So(app.Application[1].Instance[0].Port.Enabled, ShouldEqual, true)
						So(app.Application[1].Instance[0].Port.Value, ShouldEqual, 11102)
					})
					Convey("Check application[1] instance securePort data", func() {
						So(app.Application[1].Instance[0].SecurePort, ShouldNotBeEmpty)
						So(app.Application[1].Instance[0].SecurePort.Enabled, ShouldEqual, false)
						So(app.Application[1].Instance[0].SecurePort.Value, ShouldEqual, 7002)
					})

					So(app.Application[1].Instance[0].CountryId, ShouldEqual, 1)
					Convey("Check application[1] instance dataCenterInfo data", func() {
						So(app.Application[1].Instance[0].DataCenterInfo, ShouldNotBeEmpty)
						So(app.Application[1].Instance[0].DataCenterInfo.Name, ShouldEqual, model.MyOwn)
						So(app.Application[1].Instance[0].DataCenterInfo.Class, ShouldEqual, "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo")
					})
					Convey("Check application[1] instance LeaseInfo data", func() {
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
					So(app.Application[1].Instance[0].ActionType, ShouldEqual, model.ADDED)
				})
			})
		})
	})
}

// 测试 xml格式数据转对象再转json测试
func TestParseXml2Json(t *testing.T) {
	app := new(model.Applications)
	app2 := new(model.Applications)
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

func initClient() {
	goeureka.New(goeureka.AppInfo{"SPRING-CLOUD-EUREKA-CLIENT", "192.168.199.239", 8791, "MAX.lan:spring-cloud-eureka-client:8791", ""})
}

var DefaultTransport http.RoundTripper = &MockTransport{}

type MockTransport struct {
	Transport *http.Transport
}

func (t MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	logger.Debug("mock transport!!!")
	var buffer bytes.Buffer
	buffer.WriteString("HTTP/1.1 200\n")
	buffer.WriteString("Content-Type: application/xml\n")
	time.LoadLocation("Local")
	buffer.WriteString("Date:")
	buffer.WriteString(time.Now().String())
	buffer.WriteString("\n")
	buffer.WriteString("\n")

	if req.URL.RequestURI() == "/eureka/apps" {
		logger.Debug("/eureka/apps")
		//quote := strconv.AppendQuoteToASCII([]byte(resHeader), xmldata)
		//buf := strings.NewReader(resHeader)
		//buf := bytes.NewBuffer(quote)
		//buffer.WriteString(resHeader)
		buffer.WriteString(xmldata)
		buf := bytes.NewReader(buffer.Bytes())
		return http.ReadResponse(bufio.NewReader(buf), req)
	}

	return t.Transport.RoundTrip(req)
}

// mock测试client端
func TestMockClient(t *testing.T) {
	u, err := url.Parse("http://localhost:8761/eureka/apps")
	if err != nil {
		t.Error(err)
	}
	resp, err := DefaultTransport.RoundTrip(&http.Request{
		URL:        u,
		Method:     "GET",
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	})
	if err != nil {
		t.Error(err)
		return
	}

	apps := new(model.Applications)
	xml.NewDecoder(resp.Body).Decode(apps)
	checkAppData(t, *apps)
}

// 使用gock测试-查询所有实例
func TestEurekaClient_GetApps(t *testing.T) {
	defer gock.Off()
	initClient()

	gock.New(goeureka.Client.AppInfo.EurekaURL).
		Get("/eureka/apps").
		Reply(200).XML(xmldata)
	apps := goeureka.Client.GetApps()
	checkAppData(t, apps)
}
