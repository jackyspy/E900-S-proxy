package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
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

	// proxy.OnRequest(goproxy.ReqHostIs("PROXY")).DoFunc(
	// 	func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	// 		filename := path.Join(".", ctx.Req.URL.Path)
	// 		fmt.Println(filename)
	// 		if strings.Contains(filename, "..") {
	// 			return r, notFound(r)
	// 		}

	// 		if _, err := os.Stat(filename); os.IsNotExist(err) {
	// 			return r, notFound(r)
	// 		}

	// 		bytes, err := ioutil.ReadFile(filename)
	// 		if err != nil {
	// 			ctx.Warnf("%s", err)
	// 			return r, notFound(r)
	// 		}

	// 		return r, goproxy.NewResponse(r, "application/octet-stream", http.StatusOK, string(bytes))
	// 	})

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
		Url:      "",
		Version:  "3.02.008",
	}
}

func notFound(r *http.Request) *http.Response {
	return goproxy.NewResponse(r, "text/plain", http.StatusNotFound, "Not Found")
}

func getAppUrls(count int) []AppUrl {
	var result []AppUrl
	c := 0

	for _, x := range getAppUrlsFromApks() {
		result = append(result, x)
		if c++; c == count {
			return result
		}
	}

	for _, x := range getAppUrlsFromFile() {
		result = append(result, x)
		if c++; c == count {
			return result
		}
	}

	for ; c < count; c++ {
		result = append(result, newAppUrlNoUpdate())
	}

	return result
}

func quote(s string) string {
	return (&url.URL{Path: s}).RequestURI()
}

func getAppUrlsFromApks() []AppUrl {
	var result []AppUrl

	files, err := ioutil.ReadDir(".")
	if err != nil {
		return result
	}

	for _, file := range files {
		filename := strings.ToLower(file.Name())
		if file.IsDir() || !strings.HasSuffix(filename, ".apk") {
			continue
		}

		result = append(result, newAppUrl(fmt.Sprintf("http://PROXY/%s", quote(filename))))
	}

	return result
}

func getAppUrlsFromFile() []AppUrl {
	var result []AppUrl

	f, err := os.Open("apps.txt")
	if err != nil {
		return result
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	for {
		line, err := rd.ReadString('\n')
		if err != nil && io.EOF != err {
			break
		}

		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			result = append(result, newAppUrl(line))
		}

		if io.EOF == err {
			break
		}
	}

	return result
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

