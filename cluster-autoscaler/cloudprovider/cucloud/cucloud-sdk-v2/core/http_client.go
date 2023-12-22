/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core/auth"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core/response"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core/util"
)

const (
	HeaderContentType          = "Content-Type"
	HeaderJSONContentTypeValue = "application/json"
)

type HttpClient struct {
	baseUrl     string
	credential  *auth.Credential
	httpClient  *http.Client

	body    map[string]interface{}
	headers map[string]string
}

func NewHttpClient(credential *auth.Credential) *HttpClient {
	client := &HttpClient{
		baseUrl:     credential.Endpoint,
		credential:  credential,
		body:        make(map[string]interface{}),
		headers:     make(map[string]string),
	}

	client.httpClient = http.DefaultClient

	return client
}

func (c *HttpClient) do(method string) (*response.Response, error) {
	bodyBuf := &bytes.Buffer{}
	err := json.NewEncoder(bodyBuf).Encode(c.body)
	if err != nil {
		return nil, err
	}
	baseUrl := c.baseUrl
	request, err := http.NewRequest(method, baseUrl, bodyBuf)
	if err != nil {
		return nil, err
	}
	//构造请求头参数
	request.Header[auth.HeaderAlg] = []string{auth.HeaderAlgValue}
	request.Header[auth.HeaderAk] = []string{c.credential.AccessKey}
	// 签名标识
	request.Header[auth.HeaderSigned] = []string{auth.SignedHeaderValue}
	requestTime := time.Now().UnixNano() / 1e6
	request.Header[auth.HeaderReqTime] = []string{strconv.FormatInt(requestTime, 10)}
	// 赋值签名需要的body
	signBody := c.body
	// get接口 query参数参与加签, 其他参数只取body参数
	if method == http.MethodGet {
		index := strings.Index(baseUrl, "?")
		if index > 0 {
			// 请求地址中包含参数，统一放在body中
			urlParam := baseUrl[index+1:]
			split := strings.Split(urlParam, "&")
			for _, s := range split {
				p := strings.Split(s, "=")
				signBody[p[0]] = p[1]
			}
		}
	}
	//参数加签，生成签名字符串
	sign, err := auth.GenerateSign(signBody, c.credential.SecretKey, request.Header)
	if err != nil {
		return nil, err
	}
	request.Header[auth.HeaderSign] = []string{sign}
	// 自定义请求头
	if c.headers != nil {
		for k, v := range c.headers {
			request.Header[k] = []string{v}
		}
	}

	r := response.Response{}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err = response.CheckResponse(resp, &r); err != nil {
		return nil, err
	}

	return &r, err
}

func (c *HttpClient) Get() (*response.Response, error) {
	return c.do(http.MethodGet)
}

func (c *HttpClient) Delete() (*response.Response, error) {
	c.headers[HeaderContentType] = HeaderJSONContentTypeValue
	return c.do(http.MethodDelete)
}

func (c *HttpClient) Post() (*response.Response, error) {
	c.headers[HeaderContentType] = HeaderJSONContentTypeValue
	return c.do(http.MethodPost)
}

func (c *HttpClient) Uri(uri string) *HttpClient {
	c.baseUrl = util.GenUrl(c.credential.Endpoint, uri)
	return c
}

func (c *HttpClient) WithRawBody(raw interface{}) *HttpClient {
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &c.body)
	return c
}

func (c *HttpClient) WithHeaders(headers map[string]string) *HttpClient {
	c.headers = headers
	return c
}
