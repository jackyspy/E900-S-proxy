package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/miekg/dns"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var (
	localIP string
)

type AppUrl struct {
	AppName  string `json:"appName"`
	FileName string `json:"fileName"`
	IsUpdate bool   `json:"isUpdate"`
	MD5      string `json:"md5"`
	PkgName  string `json:"pkgName"`
	Url      string `json:"url"`
	Version  string `json:"version"`
}

type AppResult struct {
	Code        int      `json:"code"`
	Description string   `json:"description"`
	AppURL      []AppUrl `json:"appURL"`
}

func NewProxy() *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest(goproxy.UrlHasPrefix("appStoreRrc.cnitv.net:8090/tv/updater2")).DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			r.ParseForm()
			appCount := strings.Count(r.Form["applist"][0], "{")
			data, _ := json.Marshal([]interface{}{
				map[string]interface{}{
					"result": AppResult{
						Code:        0,
						Description: "成功获取升级信息",
						AppURL:      getAppUrls(appCount),
					},
				},
			})

			fmt.Println(string(data))

			return r, goproxy.NewResponse(r, "text/html;charset=utf-8", http.StatusOK, string(data))
		})

	return proxy
}

func newAppUrl(url string) AppUrl {
	return AppUrl{
		AppName:  "语音助手core",
		FileName: "aiCore_YueMe.apk",
		IsUpdate: true,
		MD5:      "1efd99d6ee5d010f369c9a9b83752102",
		PkgName:  "com.keylab.speech.core.yueme",
		Url:      url,
		Version:  "3.02.008",
	}
}

func newAppUrlNoUpdate() AppUrl {
	return AppUrl{
		IsUpdate: false,
		PkgName:  "com.keylab.speech.core.yueme",
		Version:  "3.02.008",
	}
}

func getAppUrls(count int) []AppUrl {
	result := []AppUrl{}

	files, err := ioutil.ReadDir(".")
	if err == nil {
		for _, file := range files {
			filename := strings.ToLower(file.Name())
			if file.IsDir() || !strings.HasSuffix(filename, ".apk") {
				continue
			}

			result = append(result, newAppUrl(fmt.Sprintf("http://appStoreRrc.cnitv.net:8090/_apks/%s", quote(filename))))
		}
	}

	for c := len(result); c < count; c++ {
		result = append(result, newAppUrlNoUpdate())
	}

	return result
}

func quote(s string) string {
	return (&url.URL{Path: s}).RequestURI()
}

type muxHandler struct {
	proxyHandler      http.Handler
	staticFileHandler http.Handler
}

func (th *muxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host == "appStoreRrc.cnitv.net:8090" && strings.HasPrefix(r.URL.Path, "/_apks/") {
		r.URL.Path = r.URL.Path[6:]
		th.staticFileHandler.ServeHTTP(w, r)
		return
	}

	r.URL.Scheme = "http"
	r.URL.Host = r.Host
	th.proxyHandler.ServeHTTP(w, r)
}

func UnFqdn(s string) string {
	if dns.IsFqdn(s) {
		return s[:len(s)-1]
	}
	return s
}

func doUDP(w dns.ResponseWriter, req *dns.Msg) {
	if len(req.Question) == 0 {
		dns.HandleFailed(w, req)
		return
	}

	q := req.Question[0]
	if q.Name == "appStoreRrc.cnitv.net." {
		m := new(dns.Msg)
		m.SetReply(req)

		rr_header := dns.RR_Header{
			Name:   q.Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    600,
		}
		m.Answer = append(m.Answer, &dns.A{rr_header, net.ParseIP(localIP).To4()})

		w.WriteMsg(m)
		return
	}

	c := &dns.Client{Net: "udp"}
	resp, _, err := c.Exchange(req, "114.114.114.114:53")
	if err != nil {
		dns.HandleFailed(w, req)
		return
	}
	w.WriteMsg(resp)
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "114.114.114.114:53")
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	defer conn.Close()

	return strings.Split(conn.LocalAddr().String(), ":")[0]
}

func init() {
	localIP = getLocalIP()
}

func main() {
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	flag.Parse()

	udpServer := &dns.Server{Addr: ":53", Net: "udp"}
	dns.HandleFunc(".", doUDP)
	go udpServer.ListenAndServe()

	proxy := NewProxy()
	proxy.Verbose = *verbose
	mux := muxHandler{proxyHandler: proxy, staticFileHandler: http.FileServer(http.Dir("./"))}
	log.Fatal(http.ListenAndServe(":8090", &mux))
}
