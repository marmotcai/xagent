package kubernetes

import (
	"context"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/kubernetes/object"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
	api "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWildcard(t *testing.T) {
	var tests = []struct {
		s        string
		expected bool
	}{
		{"mynamespace", false},
		{"*", true},
		{"any", true},
		{"my*space", false},
		{"*space", false},
		{"myname*", false},
	}

	for _, te := range tests {
		got := wildcard(te.s)
		if got != te.expected {
			t.Errorf("Expected Wildcard result '%v' for example '%v', got '%v'.", te.expected, te.s, got)
		}
	}
}

func TestEndpointHostname(t *testing.T) {
	var tests = []struct {
		ip               string
		hostname         string
		expected         string
		podName          string
		endpointNameMode bool
	}{
		{"10.11.12.13", "", "10-11-12-13", "", false},
		{"10.11.12.13", "epname", "epname", "", false},
		{"10.11.12.13", "", "10-11-12-13", "hello-abcde", false},
		{"10.11.12.13", "epname", "epname", "hello-abcde", false},
		{"10.11.12.13", "epname", "epname", "hello-abcde", true},
		{"10.11.12.13", "", "hello-abcde", "hello-abcde", true},
	}
	for _, test := range tests {
		result := endpointHostname(object.EndpointAddress{IP: test.ip, Hostname: test.hostname, TargetRefName: test.podName}, test.endpointNameMode)
		if result != test.expected {
			t.Errorf("Expected endpoint name for (ip:%v hostname:%v) to be '%v', but got '%v'", test.ip, test.hostname, test.expected, result)
		}
	}
}

type APIConnServiceTest struct{}

func (APIConnServiceTest) HasSynced() bool                           { return true }
func (APIConnServiceTest) Run()                                      { return }
func (APIConnServiceTest) Stop() error                               { return nil }
func (APIConnServiceTest) PodIndex(string) []*object.Pod             { return nil }
func (APIConnServiceTest) SvcIndexReverse(string) []*object.Service  { return nil }
func (APIConnServiceTest) EpIndexReverse(string) []*object.Endpoints { return nil }
func (APIConnServiceTest) Modified() int64                           { return 0 }

func (APIConnServiceTest) SvcIndex(string) []*object.Service {
	svcs := []*object.Service{
		{
			Name:      "svc1",
			Namespace: "testns",
			ClusterIP: "10.0.0.1",
			Ports: []api.ServicePort{
				{Name: "http", Protocol: "tcp", Port: 80},
			},
		},
		{
			Name:      "hdls1",
			Namespace: "testns",
			ClusterIP: api.ClusterIPNone,
		},
		{
			Name:         "external",
			Namespace:    "testns",
			ExternalName: "coredns.io",
			Type:         api.ServiceTypeExternalName,
			Ports: []api.ServicePort{
				{Name: "http", Protocol: "tcp", Port: 80},
			},
		},
	}
	return svcs
}

func (APIConnServiceTest) ServiceList() []*object.Service {
	svcs := []*object.Service{
		{
			Name:      "svc1",
			Namespace: "testns",
			ClusterIP: "10.0.0.1",
			Ports: []api.ServicePort{
				{Name: "http", Protocol: "tcp", Port: 80},
			},
		},
		{
			Name:      "hdls1",
			Namespace: "testns",
			ClusterIP: api.ClusterIPNone,
		},
		{
			Name:         "external",
			Namespace:    "testns",
			ExternalName: "coredns.io",
			Type:         api.ServiceTypeExternalName,
			Ports: []api.ServicePort{
				{Name: "http", Protocol: "tcp", Port: 80},
			},
		},
	}
	return svcs
}

func (APIConnServiceTest) EpIndex(string) []*object.Endpoints {
	eps := []*object.Endpoints{
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "172.0.0.1", Hostname: "ep1a"},
					},
					Ports: []object.EndpointPort{
						{Port: 80, Protocol: "tcp", Name: "http"},
					},
				},
			},
			Name:      "svc1",
			Namespace: "testns",
		},
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "172.0.0.2"},
					},
					Ports: []object.EndpointPort{
						{Port: 80, Protocol: "tcp", Name: "http"},
					},
				},
			},
			Name:      "hdls1",
			Namespace: "testns",
		},
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "172.0.0.3"},
					},
					Ports: []object.EndpointPort{
						{Port: 80, Protocol: "tcp", Name: "http"},
					},
				},
			},
			Name:      "hdls1",
			Namespace: "testns",
		},
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "10.9.8.7", NodeName: "test.node.foo.bar"},
					},
				},
			},
		},
	}
	return eps
}

func (APIConnServiceTest) EndpointsList() []*object.Endpoints {
	eps := []*object.Endpoints{
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "172.0.0.1", Hostname: "ep1a"},
					},
					Ports: []object.EndpointPort{
						{Port: 80, Protocol: "tcp", Name: "http"},
					},
				},
			},
			Name:      "svc1",
			Namespace: "testns",
		},
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "172.0.0.2"},
					},
					Ports: []object.EndpointPort{
						{Port: 80, Protocol: "tcp", Name: "http"},
					},
				},
			},
			Name:      "hdls1",
			Namespace: "testns",
		},
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "172.0.0.3"},
					},
					Ports: []object.EndpointPort{
						{Port: 80, Protocol: "tcp", Name: "http"},
					},
				},
			},
			Name:      "hdls1",
			Namespace: "testns",
		},
		{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{IP: "10.9.8.7", NodeName: "test.node.foo.bar"},
					},
				},
			},
		},
	}
	return eps
}

func (APIConnServiceTest) GetNodeByName(name string) (*api.Node, error) {
	return &api.Node{
		ObjectMeta: meta.ObjectMeta{
			Name: "test.node.foo.bar",
		},
	}, nil
}

func (APIConnServiceTest) GetNamespaceByName(name string) (*api.Namespace, error) {
	return &api.Namespace{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
	}, nil
}

func TestServices(t *testing.T) {
	k := New([]string{"interwebs.test."})
	k.APIConn = &APIConnServiceTest{}

	type svcAns struct {
		host string
		key  string
	}
	type svcTest struct {
		qname  string
		qtype  uint16
		answer svcAns
	}
	tests := []svcTest{
		// Cluster IP Services
		{qname: "svc1.testns.svc.interwebs.test.", qtype: dns.TypeA, answer: svcAns{host: "10.0.0.1", key: "/" + coredns + "/test/interwebs/svc/testns/svc1"}},
		{qname: "_http._tcp.svc1.testns.svc.interwebs.test.", qtype: dns.TypeSRV, answer: svcAns{host: "10.0.0.1", key: "/" + coredns + "/test/interwebs/svc/testns/svc1"}},
		{qname: "ep1a.svc1.testns.svc.interwebs.test.", qtype: dns.TypeA, answer: svcAns{host: "172.0.0.1", key: "/" + coredns + "/test/interwebs/svc/testns/svc1/ep1a"}},

		// External Services
		{qname: "external.testns.svc.interwebs.test.", qtype: dns.TypeCNAME, answer: svcAns{host: "coredns.io", key: "/" + coredns + "/test/interwebs/svc/testns/external"}},
	}

	for i, test := range tests {
		state := request.Request{
			Req:  &dns.Msg{Question: []dns.Question{{Name: test.qname, Qtype: test.qtype}}},
			Zone: "interwebs.test.", // must match from k.Zones[0]
		}
		svcs, e := k.Services(context.TODO(), state, false, plugin.Options{})
		if e != nil {
			t.Errorf("Test %d: got error '%v'", i, e)
			continue
		}
		if len(svcs) != 1 {
			t.Errorf("Test %d, expected 1 answer, got %v", i, len(svcs))
			continue
		}

		if test.answer.host != svcs[0].Host {
			t.Errorf("Test %d, expected host '%v', got '%v'", i, test.answer.host, svcs[0].Host)
		}
		if test.answer.key != svcs[0].Key {
			t.Errorf("Test %d, expected key '%v', got '%v'", i, test.answer.key, svcs[0].Key)
		}
	}
}

func TestServicesAuthority(t *testing.T) {
	k := New([]string{"interwebs.test."})
	k.APIConn = &APIConnServiceTest{}

	type svcAns struct {
		host string
		key  string
	}
	type svcTest struct {
		interfaceAddrs func() net.IP
		qname          string
		qtype          uint16
		answer         *svcAns
	}
	tests := []svcTest{
		{interfaceAddrs: func() net.IP { return net.ParseIP("127.0.0.1") }, qname: "ns.dns.interwebs.test.", qtype: dns.TypeA, answer: &svcAns{host: "127.0.0.1", key: "/" + coredns + "/test/interwebs/dns/ns"}},
		{interfaceAddrs: func() net.IP { return net.ParseIP("127.0.0.1") }, qname: "ns.dns.interwebs.test.", qtype: dns.TypeAAAA},
		{interfaceAddrs: func() net.IP { return net.ParseIP("::1") }, qname: "ns.dns.interwebs.test.", qtype: dns.TypeA},
		{interfaceAddrs: func() net.IP { return net.ParseIP("::1") }, qname: "ns.dns.interwebs.test.", qtype: dns.TypeAAAA, answer: &svcAns{host: "::1", key: "/" + coredns + "/test/interwebs/dns/ns"}},
	}

	for i, test := range tests {
		k.interfaceAddrsFunc = test.interfaceAddrs

		state := request.Request{
			Req:  &dns.Msg{Question: []dns.Question{{Name: test.qname, Qtype: test.qtype}}},
			Zone: "interwebs.test.", // must match from k.Zones[0]
		}
		svcs, e := k.Services(context.TODO(), state, false, plugin.Options{})
		if e != nil {
			t.Errorf("Test %d: got error '%v'", i, e)
			continue
		}
		if test.answer != nil && len(svcs) != 1 {
			t.Errorf("Test %d, expected 1 answer, got %v", i, len(svcs))
			continue
		}
		if test.answer == nil && len(svcs) != 0 {
			t.Errorf("Test %d, expected no answer, got %v", i, len(svcs))
			continue
		}

		if test.answer == nil && len(svcs) == 0 {
			continue
		}

		if test.answer.host != svcs[0].Host {
			t.Errorf("Test %d, expected host '%v', got '%v'", i, test.answer.host, svcs[0].Host)
		}
		if test.answer.key != svcs[0].Key {
			t.Errorf("Test %d, expected key '%v', got '%v'", i, test.answer.key, svcs[0].Key)
		}
	}
}

func TestServiceFQDN(t *testing.T) {
	fqdn := serviceFQDN(
		&object.Service{
			Name:      "svc1",
			Namespace: "testns",
		}, "cluster.local")

	expected := "svc1.testns.svc.cluster.local."
	if fqdn != expected {
		t.Errorf("Expected '%v', got '%v'.", expected, fqdn)
	}
}

func TestPodFQDN(t *testing.T) {
	fqdn := podFQDN(
		&object.Pod{
			Name:      "pod1",
			Namespace: "testns",
			PodIP:     "10.10.0.10",
		}, "cluster.local")

	expected := "10-10-0-10.testns.pod.cluster.local."
	if fqdn != expected {
		t.Errorf("Expected '%v', got '%v'.", expected, fqdn)
	}
	fqdn = podFQDN(
		&object.Pod{
			Name:      "pod1",
			Namespace: "testns",
			PodIP:     "aaaa:bbbb:cccc::zzzz",
		}, "cluster.local")

	expected = "aaaa-bbbb-cccc--zzzz.testns.pod.cluster.local."
	if fqdn != expected {
		t.Errorf("Expected '%v', got '%v'.", expected, fqdn)
	}
}

func TestEndpointFQDN(t *testing.T) {
	fqdns := endpointFQDN(
		&object.Endpoints{
			Subsets: []object.EndpointSubset{
				{
					Addresses: []object.EndpointAddress{
						{
							IP:       "172.0.0.1",
							Hostname: "ep1a",
						},
						{
							IP: "172.0.0.2",
						},
					},
				},
			},
			Name:      "svc1",
			Namespace: "testns",
		}, "cluster.local", false)

	expected := []string{
		"ep1a.svc1.testns.svc.cluster.local.",
		"172-0-0-2.svc1.testns.svc.cluster.local.",
	}

	for i := range fqdns {
		if fqdns[i] != expected[i] {
			t.Errorf("Expected '%v', got '%v'.", expected[i], fqdns[i])
		}
	}
}
