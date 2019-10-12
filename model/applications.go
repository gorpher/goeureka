package model

type ActionType string

const (
	ADDED    ActionType = "ADDED"
	MODIFIED            = "MODIFIED"
	DELETED             = "DELETED"
)

type InstanceStatus string

const (
	UP             InstanceStatus = "UP"
	DOWN                          = "DOWN"
	STARTING                      = "STARTING"
	OUT_OF_SERVICE                = "OUT_OF_SERVICE"
	UNKNOWN                       = "UNKNOWN"
)

type DataCenterInfoName string

type DataCenterInfo struct {
	Class string `json:"@class" xml:"class,attr"`
	Name  string `json:"name" xml:"name"`
}

const (
	Netflix DataCenterInfoName = "Netflix"
	Amazon                     = "Amazon"
	MyOwn                      = "MyOwn"
)

type LeaseInfo struct {
	RenewalIntervalInSecs int   `json:"renewalIntervalInSecs" xml:"renewalIntervalInSecs"` //30
	DurationInSecs        int   `json:"durationInSecs" xml:"durationInSecs"`               //90
	RegistrationTimestamp int64 `json:"registrationTimestamp" xml:"registrationTimestamp"`
	LastRenewalTimestamp  int64 `json:"lastRenewalTimestamp" xml:"lastRenewalTimestamp"`
	EvictionTimestamp     int64 `json:"evictionTimestamp" xml:"evictionTimestamp"`
	ServiceUpTimestamp    int64 `json:"serviceUpTimestamp" xml:"serviceUpTimestamp"`
}

type Port struct {
	Enabled bool `json:"@enabled" xml:"enabled,attr"`
	Value   int  `json:"$" xml:",chardata"`
}

type Instance struct {
	InstanceID                    string         `json:"instanceId" xml:"instanceId"`
	App                           string         `json:"app" xml:"app"`
	HostName                      string         `json:"hostName" xml:"hostName"`
	IpAddr                        string         `json:"ipAddr" xml:"ipAddr"`
	Sid                           string         `json:"sid" xml:"sid"` //default:na
	HomePageUrl                   string         `json:"homePageUrl" xml:"homePageUrl"`
	StatusPageUrl                 string         `json:"statusPageUrl" xml:"statusPageUrl"`
	HealthCheckUrl                string         `json:"healthCheckUrl" xml:"healthCheckUrl"`
	VipAddress                    string         `json:"vipAddress" xml:"vipAddress"`
	SecureVipAddress              string         `json:"secureVipAddress" xml:"secureVipAddress"`
	CountryId                     int            `json:"countryId" xml:"countryId"` // 	US:1
	DataCenterInfo                DataCenterInfo `json:"dataCenterInfo" xml:"dataCenterInfo"`
	Status                        InstanceStatus `json:"status" xml:"status"`
	Overriddenstatus              InstanceStatus `json:"overriddenstatus" xml:"overriddenstatus"`
	LeaseInfo                     LeaseInfo      `json:"leaseInfo" xml:"leaseInfo"`
	IsCoordinatingDiscoveryServer bool           `json:"isCoordinatingDiscoveryServer" xml:"isCoordinatingDiscoveryServer"`
	LastUpdatedTimestamp          int64          `json:"lastUpdatedTimestamp" xml:"lastUpdatedTimestamp"`
	LastDirtyTimestamp            int64          `json:"lastDirtyTimestamp" xml:"lastDirtyTimestamp"`
	ActionType                    ActionType     `json:"actionType" xml:"actionType"`
	Port                          Port           `json:"port" xml:"port"`             //7001 $7001 @enabled=true
	SecurePort                    Port           `json:"securePort" xml:"securePort"` //7002 $7002 @enabled=false
	Metadata                      interface{}    `json:"metadata,omitempty" xml:"metadata"`
}

type Application struct {
	Name     string     `json:"name" xml:"name"`
	Instance []Instance `json:"instance" xml:"instance"`
}

type Applications struct {
	AppsHashCode string        `json:"apps__hashcode" xml:"apps__hashcode"`
	VersionDelta string        `json:"versions__delta" xml:"versions__delta"`
	Application  []Application `json:"application" xml:"application"`
}
