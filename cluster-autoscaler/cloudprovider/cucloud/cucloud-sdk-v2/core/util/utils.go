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

package util

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"
)

func GenUrl(endpoint, path string) string {
	u, _ := url.Parse(endpoint)
	rel, _ := url.Parse(path)
	u = u.ResolveReference(rel)
	return u.String()
}

func GetStrValue(data interface{}) string {
	return fmt.Sprintf("%v", data)
}

func GetFloatValue(val string) float64 {
	float, _ := strconv.ParseFloat(val, 64)
	return float
}

func GetBoolPtr(val bool) *bool {
	return &val
}

func TransferToMap(data interface{}) map[string]interface{} {
	if m, ok := data.(map[string]interface{}); ok {
		return m
	}
	m := make(map[string]interface{})
	bytes, _ := json.Marshal(data)
	_ = json.Unmarshal(bytes, &m)
	return m
}

func TransferToMapWithTag(data interface{}, tagName string) map[string]interface{} {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	m := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		// 过滤私有的,不导出
		if t.Field(i).PkgPath != "" {
			continue
		}
		if tag, have := t.Field(i).Tag.Lookup(tagName); have {
			switch v.Field(i).Kind() {
			case reflect.Struct:
				valueValue := v.Field(i).Interface()
				m[tag] = TransferToMapWithTag(valueValue, tagName)
			default:
				m[tag] = v.Field(i).Interface()
			}
		}
	}
	return m
}

func getEnvOrDefault(key, defaultValue string) string {
	value, found := os.LookupEnv(key)
	if found {
		return value
	}
	return defaultValue
}
