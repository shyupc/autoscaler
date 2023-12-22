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

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
)

const (
	HeaderAlg         = "algorithm"
	HeaderAlgValue    = "HmacSHA256"
	HeaderAk          = "accessKey"
	HeaderReqTime     = "requestTime"
	HeaderSign        = "sign"
	HeaderSigned      = "signedHeader"
	SignedHeaderValue = "1"
)

func GenerateSign(body map[string]interface{}, secretKey string, header http.Header) (string, error) {
	if secretKey == "" {
		return "", errors.New("generate sign parameters error: secretKey must not be null")
	}
	// 准备加签的参数
	param := make(map[string]interface{})
	for k, v := range body {
		param[k] = v
	}
	if v, ok := header[HeaderSigned]; ok && v[0] == SignedHeaderValue {
		for k, v := range header {
			if k == HeaderSign {
				continue
			}
			param[k] = v[0]
		}
	} else {
		param[HeaderReqTime] = header[HeaderReqTime][0]
	}

	var sortedKey sort.StringSlice
	for key := range param {
		sortedKey = append(sortedKey, key)
	}
	sort.Sort(sortedKey)

	var stringToSign string
	// 按照上述key的顺序把Key，Value值拼接
	// 得到的sign值应该为key1=xxx&key2=xxx&key3=xxx
	for _, key := range sortedKey {
		kvStr := getKvStr(key, param[key])
		stringToSign += fmt.Sprintf("&%s", kvStr)
	}
	stringToSign = stringToSign[1:]
	//fmt.Printf("[generateSign] Parameters to be signed: %s\n", stringToSign)

	s := []byte(stringToSign)
	key := []byte(secretKey)
	m := hmac.New(sha256.New, key)
	m.Write(s)
	signature := hex.EncodeToString(m.Sum(nil))
	//fmt.Printf("[generateSign] Signature: %s\n", signature)

	return signature, nil
}

func getKvStr(key string, val interface{}) string {
	var kvStr string
	switch v := val.(type) {
	case string:
		kvStr = fmt.Sprintf("%s=\"%s\"", key, v)
	case float64, float32, int, int8, int16, int32, int64:
		kvStr = fmt.Sprintf("%s=%v", key, v)
	default:
		jsonVal, _ := json.Marshal(v)
		kvStr = fmt.Sprintf("%s=%s", key, string(jsonVal))
	}

	return kvStr
}

func VerifySign(body map[string]interface{}, header http.Header, secretKey string) bool {
	signString := header[HeaderSign][0]
	fmt.Printf("sign: %s\n", signString)
	encryptContent, err := GenerateSign(body, secretKey, header)
	if err != nil {
		return false
	}
	fmt.Printf("encryptSign: %s\n", encryptContent)
	if signString != encryptContent {
		fmt.Println("verify sign failed")
		return false
	}
	return true
}
