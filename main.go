package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
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

			result = append(result, newAppUrl(fmt.Sprintf("http://PROXY/%s", quote(filename))))
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
	proxyHandler   http.Handler
	defaultHandler http.Handler
}

func (th *muxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host == "PROXY" {
		th.defaultHandler.ServeHTTP(w, r)
		return
	}

	th.proxyHandler.ServeHTTP(w, r)
}

func main() {
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", ":8080", "proxy listen address")
	flag.Parse()
	proxy := NewProxy()
	proxy.Verbose = *verbose
	mux := muxHandler{proxyHandler: proxy, defaultHandler: http.FileServer(http.Dir("./"))}
	log.Fatal(http.ListenAndServe(*addr, &mux))
}
