// Copyright 2017 Istio Authors
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

package kube

import (
	"testing"

	"istio.io/pilot/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/pkg/api/v1"
)

var (
	domainSuffix = "company.com"

	camelKabobs = []struct{ in, out string }{
		{"ExampleNameX", "example-name-x"},
		{"example1", "example1"},
		{"exampleXY", "example-x-y"},
	}

	protocols = []struct {
		name  string
		proto v1.Protocol
		out   model.Protocol
	}{
		{"", v1.ProtocolTCP, model.ProtocolTCP},
		{"http", v1.ProtocolTCP, model.ProtocolHTTP},
		{"http-test", v1.ProtocolTCP, model.ProtocolHTTP},
		{"http", v1.ProtocolUDP, model.ProtocolUDP},
		{"httptest", v1.ProtocolTCP, model.ProtocolTCP},
		{"https", v1.ProtocolTCP, model.ProtocolHTTPS},
		{"https-test", v1.ProtocolTCP, model.ProtocolHTTPS},
		{"http2", v1.ProtocolTCP, model.ProtocolHTTP2},
		{"http2-test", v1.ProtocolTCP, model.ProtocolHTTP2},
		{"grpc", v1.ProtocolTCP, model.ProtocolGRPC},
		{"grpc-test", v1.ProtocolTCP, model.ProtocolGRPC},
	}
)

func TestCamelKabob(t *testing.T) {
	for _, tt := range camelKabobs {
		s := camelCaseToKabobCase(tt.in)
		if s != tt.out {
			t.Errorf("camelCaseToKabobCase(%q) => %q, want %q", tt.in, s, tt.out)
		}
	}
}

func TestConvertProtocol(t *testing.T) {
	for _, tt := range protocols {
		out := convertProtocol(tt.name, tt.proto)
		if out != tt.out {
			t.Errorf("convertProtocol(%q, %q) => %q, want %q", tt.name, tt.proto, out, tt.out)
		}
	}
}

func TestDecodeIngressRuleName(t *testing.T) {
	cases := []struct {
		ingressName      string
		ingressNamespace string
		ruleNum          int
		pathNum          int
	}{
		{"myingress", "test", 0, 0},
		{"myingress", "default", 1, 2},
		{"my-ingress", "test-namespace", 1, 2},
		{"my-cool-ingress", "new-space", 1, 2},
	}

	for _, c := range cases {
		encoded := encodeIngressRuleName(c.ingressName, c.ingressNamespace, c.ruleNum, c.pathNum)
		ingressName, ingressNamespace, ruleNum, pathNum, err := decodeIngressRuleName(encoded)
		if err != nil {
			t.Errorf("decodeIngressRuleName(%q) => error %v", encoded, err)
		}
		if ingressName != c.ingressName || ingressNamespace != c.ingressNamespace ||
			ruleNum != c.ruleNum || pathNum != c.pathNum {
			t.Errorf("decodeIngressRuleName(%q) => (%q, %q, %d, %d), want (%q, %q, %d, %d)",
				encoded,
				ingressName, ingressNamespace, ruleNum, pathNum,
				c.ingressName, c.ingressNamespace, c.ruleNum, c.pathNum,
			)
		}
	}
}

func TestIsRegularExpression(t *testing.T) {
	cases := []struct {
		s       string
		isRegex bool
	}{
		{"/api/v1/", false},
		{"/api/v1/.*", true},
		{"/api/.*/resource", true},
		{"/api/v[1-9]/resource", true},
		{"/api/.*/.*", true},
	}

	for _, c := range cases {
		if isRegularExpression(c.s) != c.isRegex {
			t.Errorf("isRegularExpression(%q) => %v, want %v", c.s, !c.isRegex, c.isRegex)
		}
	}
}

func TestServiceConversion(t *testing.T) {
	serviceName := "service1"
	namespace := "default"
	ip := "10.0.0.1"

	localSvc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			ClusterIP: ip,
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Port:     8080,
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "https",
					Protocol: v1.ProtocolTCP,
					Port:     443,
				},
			},
		},
	}

	service := convertService(localSvc, domainSuffix)
	if service == nil {
		t.Errorf("could not convert service")
	}

	if len(service.Ports) != len(localSvc.Spec.Ports) {
		t.Errorf("incorrect number of ports => %v, want %v",
			len(service.Ports), len(localSvc.Spec.Ports))
	}

	if service.External() {
		t.Error("service should not be external")
	}

	if service.Hostname != serviceHostname(serviceName, namespace, domainSuffix) {
		t.Errorf("service hostname incorrect => %q, want %q",
			service.Hostname, serviceHostname(serviceName, namespace, domainSuffix))
	}

	if service.Address != ip {
		t.Errorf("service IP incorrect => %q, want %q", service.Address, ip)
	}
}

func TestExternalServiceConversion(t *testing.T) {
	serviceName := "service1"
	namespace := "default"

	extSvc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: v1.ProtocolTCP,
				},
			},
			Type:         v1.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	service := convertService(extSvc, domainSuffix)
	if service == nil {
		t.Errorf("could not convert external service")
	}

	if len(service.Ports) != len(extSvc.Spec.Ports) {
		t.Errorf("incorrect number of ports => %v, want %v",
			len(service.Ports), len(extSvc.Spec.Ports))
	}

	if service.ExternalName != extSvc.Spec.ExternalName || !service.External() {
		t.Error("service should be external")
	}

	if service.Hostname != serviceHostname(serviceName, namespace, domainSuffix) {
		t.Errorf("service hostname incorrect => %q, want %q",
			service.Hostname, extSvc.Spec.ExternalName)
	}
}

func TestInvalidServiceConversion(t *testing.T) {
	serviceName := "service1"
	namespace := "default"

	localSvc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Port:     8080,
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "https",
					Protocol: v1.ProtocolTCP,
					Port:     443,
				},
			},
		},
	}

	if svc := convertService(localSvc, domainSuffix); svc != nil {
		t.Errorf("converted a service without a cluster IP")
	}
}

func TestInvalidExternalServiceConversion(t *testing.T) {
	serviceName := "service1"
	namespace := "default"

	extSvc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: v1.ProtocolTCP,
				},
			},
			Type: v1.ServiceTypeExternalName,
		},
	}

	if svc := convertService(extSvc, domainSuffix); svc != nil {
		t.Errorf("converted a service without an external name")
	}
}
