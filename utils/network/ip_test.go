package network

import (
	"net"
	"net/http/httptest"
	"testing"
)

func TestGetLocalIP(t *testing.T) {
	ip, err := GetLocalIP()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if ip == "" {
		t.Fatal("Expected an IP address, got an empty string")
	}
	// Optionally, check the format of the IP address
	if net.ParseIP(ip) == nil {
		t.Fatalf("Expected a valid IP address, got %s", ip)
	}
}

func TestGetRpcRegisterIP(t *testing.T) {
	expectedIP := "192.168.1.1"
	ip, err := GetRpcRegisterIP(expectedIP)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if ip != expectedIP {
		t.Fatalf("Expected %s, got %s", expectedIP, ip)
	}

	// Test with an empty string, expecting a local IP back
	ip, err = GetRpcRegisterIP("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("Expected a valid IP address, got %s", ip)
	}
	t.Log("GetRpcRegisterIP:", ip)
}

func TestGetListenIP(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", "0.0.0.0"},
		{"192.168.1.1", "192.168.1.1"},
	}

	for _, tc := range testCases {
		result := GetListenIP(tc.input)
		if result != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, result)
		}
	}
}

func TestRemoteIP(t *testing.T) {
	testCases := []struct {
		headers    map[string]string
		expectedIP string
	}{
		{map[string]string{XClientIP: "192.168.1.1"}, "192.168.1.1"},
		{map[string]string{XRealIP: "10.0.0.1"}, "10.0.0.1"},
		{map[string]string{XForwardedFor: "172.16.0.1"}, "172.16.0.1"},
		{map[string]string{}, "127.0.0.1"}, // assuming RemoteAddr is set to "::1"
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.RemoteAddr = "::1" // simulate localhost IPv6
		for key, value := range tc.headers {
			req.Header.Set(key, value)
		}

		if ip := RemoteIP(req); ip != tc.expectedIP {
			t.Errorf("Expected IP %s, got %s", tc.expectedIP, ip)
		}
	}
}
