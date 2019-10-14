package goeureka

import (
	"encoding/json"
	"encoding/xml"
	"github.com/wx11055/gone/logger"
	"gopkg.in/h2non/gock.v1"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

/**
 * Trims leading and trailing byte r from string s
 */
func trimChar(s string, r byte) string {
	sz := len(s)

	if sz > 0 && s[sz-1] == r {
		s = s[:sz-1]
	}
	sz = len(s)
	if sz > 0 && s[0] == r {
		s = s[1:sz]
	}
	return s
}

// v 接受结果的对象
func (h HttpAction) DoRequest(v interface{}) bool {
	req := buildHttpRequest(h)

	client := &http.Client{Transport: &http.Transport{}}

	//  用于gock 单元测试
	gock.InterceptClient(client)

	resp, err := client.Do(req)
	//resp, err := DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Printf("HTTP request failed: %s", err)
		return false
	}

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		d, _ := ioutil.ReadAll(resp.Body)
		logger.Error("HTTP request failed: statusCode=%d,desc: %s", resp.StatusCode, string(d))
		return false
	}
	if v == nil {
		return true
	}

	caye := resp.Header.Get("Content-Type")

	if "application/xml" == caye {
		err := xml.NewDecoder(resp.Body).Decode(v)
		if err != nil {
			log.Printf("Parser response xml data failed: %s", err)
			return false
		}
	}

	if "application/json" == caye {
		err := json.NewDecoder(resp.Body).Decode(v)
		if err != nil {
			log.Printf("Parser response json data failed: %s", err)
			return false
		}
	}

	defer resp.Body.Close()
	return true
}

func buildHttpRequest(httpAction HttpAction) *http.Request {
	var req *http.Request
	var err error
	if httpAction.Body != "" {
		reader := strings.NewReader(httpAction.Body)
		req, err = http.NewRequest(httpAction.Method, httpAction.Url, reader)
	} else if httpAction.Template != "" {
		reader := strings.NewReader(httpAction.Template)
		req, err = http.NewRequest(httpAction.Method, httpAction.Url, reader)
	} else {
		req, err = http.NewRequest(httpAction.Method, httpAction.Url, nil)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Add headers
	req.Header.Add("Accept", httpAction.Accept)
	if httpAction.ContentType != "" {
		req.Header.Add("Content-Type", httpAction.ContentType)
	}
	return req
}
