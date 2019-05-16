package proxy

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"text/template"
	"time"
)

var pac struct {
	template       *template.Template
	topLevelDomain string
	directList     string
	// Assignments and reads to directList are in different goroutines. Go
	// does not guarantee atomic assignment, so we should protect these racing
	// access.
	dLRWMutex sync.RWMutex
}

func getDirectList() string {
	pac.dLRWMutex.RLock()
	dl := pac.directList
	pac.dLRWMutex.RUnlock()
	return dl
}

func updateDirectList() {
	dl := strings.Join(siteStat.GetDirectList(), "\",\n\"")
	pac.dLRWMutex.Lock()
	pac.directList = dl
	pac.dLRWMutex.Unlock()
}

func init() {
	const pacRawTmpl = `var direct = 'DIRECT';
var httpProxy = 'PROXY {{.ProxyAddr}}; DIRECT';

var directList = [
"",
"{{.DirectDomains}}"
];

var directAcc = {};
for (var i = 0; i < directList.length; i += 1) {
	directAcc[directList[i]] = true;
}

var topLevel = {
{{.TopLevel}}
};

// hostIsIP determines whether a host address is an IP address and whether
// it is private. Currenly only handles IPv4 addresses.
function hostIsIP(host) {
	var part = host.split('.');
	if (part.length != 4) {
		return [false, false];
	}
	var n;
	for (var i = 3; i >= 0; i--) {
		if (part[i].length === 0 || part[i].length > 3) {
			return [false, false];
		}
		n = Number(part[i]);
		if (isNaN(n) || n < 0 || n > 255) {
			return [false, false];
		}
	}
	if (part[0] == '127' || part[0] == '10' || (part[0] == '192' && part[1] == '168')) {
		return [true, true];
	}
	if (part[0] == '172') {
		n = Number(part[1]);
		if (16 <= n && n <= 31) {
			return [true, true];
		}
	}
	return [true, false];
}

function host2Domain(host) {
	var arr, isIP, isPrivate;
	arr = hostIsIP(host);
	isIP = arr[0];
	isPrivate = arr[1];
	if (isPrivate) {
		return "";
	}
	if (isIP) {
		return host;
	}

	var lastDot = host.lastIndexOf('.');
	if (lastDot === -1) {
		return ""; // simple host name has no domain
	}
	// Find the second last dot
	dot2ndLast = host.lastIndexOf(".", lastDot-1);
	if (dot2ndLast === -1)
		return host;

	var part = host.substring(dot2ndLast+1, lastDot);
	if (topLevel[part]) {
		var dot3rdLast = host.lastIndexOf(".", dot2ndLast-1);
		if (dot3rdLast === -1) {
			return host;
		}
		return host.substring(dot3rdLast+1);
	}
	return host.substring(dot2ndLast+1);
}

function FindProxyForURL(url, host) {
	if (url.substring(0,4) == "ftp:")
		return direct;
	if (host.substring(0,7) == "::ffff:")
		return direct;
	if (host.indexOf(".local", host.length - 6) !== -1) {
		return direct;
	}
	var domain = host2Domain(host);
	if (host.length == domain.length) {
		return directAcc[host] ? direct : httpProxy;
	}
	return (directAcc[host] || directAcc[domain]) ? direct : httpProxy;
}
`
	var err error
	pac.template, err = template.New("pac").Parse(pacRawTmpl)
	if err != nil {
		Fatal("Internal error on generating pac file template:", err)
	}

	var buf bytes.Buffer
	for k, _ := range topLevelDomain {
		buf.WriteString(fmt.Sprintf("\t\"%s\": true,\n", k))
	}
	pac.topLevelDomain = buf.String()[:buf.Len()-2] // remove the final comma
}

// No need for content-length as we are closing connection
var pacHeader = []byte("HTTP/1.1 200 OK\r\nServer: cow-proxy\r\n" +
	"Content-Type: application/x-ns-proxy-autoconfig\r\nConnection: close\r\n\r\n")

// Different client will have different proxy URL, so generate it upon each request.
func genPAC(c *clientConn) []byte {
	buf := new(bytes.Buffer)

	hproxy, ok := c.proxy.(*httpProxy)
	if !ok {
		panic("sendPAC should only be called for http proxy")
	}

	proxyAddr := hproxy.addrInPAC
	if proxyAddr == "" {
		host, _, err := net.SplitHostPort(c.LocalAddr().String())
		// This is the only check to split host port on tcp addr's string
		// representation in COW. Keep it so we will notice if there's any
		// problem in the future.
		if err != nil {
			panic("split host port on local address error")
		}
		proxyAddr = net.JoinHostPort(host, hproxy.port)
	}

	dl := getDirectList()

	if dl == "" {
		// Empty direct domain list
		buf.Write(pacHeader)
		pacproxy := fmt.Sprintf("function FindProxyForURL(url, host) { return 'PROXY %s; DIRECT'; };",
			proxyAddr)
		buf.Write([]byte(pacproxy))
		return buf.Bytes()
	}

	data := struct {
		ProxyAddr     string
		DirectDomains string
		TopLevel      string
	}{
		proxyAddr,
		dl,
		pac.topLevelDomain,
	}

	buf.Write(pacHeader)
	if err := pac.template.Execute(buf, data); err != nil {
		errl.Println("Error generating pac file:", err)
		panic("Error generating pac file")
	}
	return buf.Bytes()
}

func initPAC() {
	// we can't control goroutine scheduling, make sure when
	// initPAC is done, direct list is updated
	updateDirectList()
	go func() {
		for {
			time.Sleep(time.Minute)
			updateDirectList()
		}
	}()
}

func sendPAC(c *clientConn) error {
	_, err := c.Write(genPAC(c))
	if err != nil {
		debug.Printf("cli(%s) error sending PAC: %s", c.RemoteAddr(), err)
	}
	return err
}
