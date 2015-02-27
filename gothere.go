package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/golang/glog"
	#"github.com/mailgun/manners"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type UrlMap map[string]string

// we use COW to handle reloading the map on SIGHUP; this avoids having to
// use an expensive mutex in the HTTP handler
var urlmap unsafe.Pointer
var defaultUrl string
var sigChan = make(chan os.Signal, 1)

// loads the URL mappings from urls.txt
func loadMap() {
	file, err := os.Open("urls.txt")
	if err != nil {
		glog.Fatalln(err)
	}
	defer file.Close()
	urls := make(UrlMap)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || len(line) < 1 {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			if _, ok := urls[key]; ok {
				glog.Warningf("duplicate key %s!", key)
			}
			urls[key] = strings.TrimSpace(parts[1])
		} else {
			glog.Warningf("skipping malformed line:\n%s", line)
		}
	}
	if err := scanner.Err(); err != nil {
		glog.Errorln(err)
	}
	glog.Infof("loaded %d mappings", len(urls))
	atomic.StorePointer(&urlmap, (unsafe.Pointer)(&urls))
}

// HTTP handler. If the requested URL is in our map,
// we redirect to the target. Otherwise we redirect to
// the default URL if specified.
func handler(w http.ResponseWriter, r *http.Request) {
	urls := *(*UrlMap)(atomic.LoadPointer(&urlmap))
	dest, ok := urls[strings.ToLower(r.URL.Path)]
	if !ok {
		glog.Warningf("no match for %s", r.URL.Path)
		if defaultUrl != "" {
			http.Redirect(w, r, defaultUrl, 302)
			return
		} else {
			http.Error(w, "not found", 404)
			return
		}
	}
	glog.Infof("%s %s %s %s", time.Now(), r.RemoteAddr, r.URL.Path, dest)
	http.Redirect(w, r, dest, 302)
}

func onexit() {
	fmt.Println("got here")
}

func main() {
	defer onexit()
	runtime.GOMAXPROCS(runtime.NumCPU())
	loadMap()
	signal.Notify(sigChan, syscall.SIGHUP)
	go func() {
		for _ = range sigChan {
			glog.Infoln("got SIGHUP; reloading urls.txt")
			loadMap()
		}
	}()

	var pPort = flag.Int("port", 80, "listening port")
	var pUrl = flag.String("defaultUrl", "http://google.com", "default URL")
	flag.Parse()

	defaultUrl = *pUrl
	http.HandleFunc("/", handler)
	glog.Infof("pid %d; listening on port %d", os.Getpid(), *pPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *pPort), nil)
	if err != nil {
		glog.Fatalln(err)
	}
}
