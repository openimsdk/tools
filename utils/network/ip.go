// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package network

import (
	"errors"
	"net"
	"net/http"
	"strings"

	_ "github.com/openimsdk/tools/errs"
)

// Define http headers.
const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
	XClientIP     = "x-client-ip"
)

func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil && isPrivateIP(ipnet.IP) {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("no suitable IP found")
}

// isPrivateIP checks if the given IP is a private address.
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		}
	}
	return false
}

func GetRpcRegisterIP(configIP string) (string, error) {
	registerIP := configIP
	if registerIP == "" {
		ip, err := GetLocalIP()
		if err != nil {
			return "", err
		}
		registerIP = ip
	}
	return registerIP, nil
}

func GetListenIP(configIP string) string {
	if configIP == "" {
		return "0.0.0.0"
	}
	return configIP
}

// RemoteIP returns the remote ip of the request.
func RemoteIP(req *http.Request) string {
	if ip := req.Header.Get(XClientIP); ip != "" {
		return ip
	} else if ip := req.Header.Get(XRealIP); ip != "" {
		return ip
	} else if ip := req.Header.Get(XForwardedFor); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		ip = req.RemoteAddr
	}

	if ip == "::1" {
		return "127.0.0.1"
	}
	return ip
}
