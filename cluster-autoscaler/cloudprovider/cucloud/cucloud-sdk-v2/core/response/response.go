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

package response

import (
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/cucloud/cucloud-sdk-v2/core/util"
	"net/http"
)

type Response struct {
	Code      interface{}            `json:"code,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Message   string                 `json:"message,omitempty"`
	RequestId string                 `json:"requestId,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Path      string                 `json:"path,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
	IsSuccess bool                   `json:"isSuccess,omitempty"`
}

func (r *Response) getCode() string {
	return util.GetStrValue(r.Code)
}

func (r *Response) Decode(dest interface{}) error {
	bytes, err := json.Marshal(r.Result)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(bytes, dest); err != nil {
		return err
	}

	return nil
}

// respError contains an error response from the server.
type respError struct {
	// StatusCode is the HTTP response status code and will always be populated.
	StatusCode int `json:"statusCode"`
	// Code is the API error code that is given in the error message.
	Code string `json:"code"`
	// Message is the server response message and is only populated when
	// explicitly referenced by the JSON server response.
	Message string `json:"message"`
	// Body is the raw response returned by the server.
	// It is often but not always JSON, depending on how the request fails.
	Body string
	// Header contains the response header fields from the server.
	Header http.Header
	// URL is the URL of the original HTTP request and will always be populated.
	URL       string
	RequestId string
}

// Error converts the Error type to a readable string.
func (e *respError) Error() string {
	// If the message is empty return early.
	msg := e.Message
	if msg == "" {
		msg = e.Body
	}

	return fmt.Sprintf("api call to [%s], requestId [%s], api router [%s], got HTTP response status code %d error code %q: %s",
		e.URL, e.RequestId, e.Header.Get("X-Gateway-Route-No"), e.StatusCode, e.Code, msg)
}

// CheckResponse returns an error (of type *respError) if the response
// status code is not 2xx.
func CheckResponse(resp *http.Response, r *Response) error {
	const MinValidStatusCode = 200
	const MaxValidStatusCode = 299

	defer resp.Body.Close()
	resBodyReads, ioReaderErr := io.ReadAll(resp.Body)
	e := &respError{
		URL:        resp.Request.URL.String(),
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       string(resBodyReads),
	}
	if ioReaderErr == nil {
		if resp.StatusCode >= MinValidStatusCode && resp.StatusCode <= MaxValidStatusCode {
			err := json.Unmarshal(resBodyReads, r)
			if err != nil {
				return err
			}
			if r.getCode() != "200" {
				e.Code = r.getCode()
				e.Message = r.Message
				e.RequestId = r.RequestId
				return e
			}
			return nil
		}
	}

	return e
}
