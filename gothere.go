package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/golang/glog"
)

type UrlMap map[string]string

// we use atomic pointer switching to handle reloading the map on SIGHUP,
// this avoids having to use a more expensive mutex wait in the HTTP handler
var urlmap unsafe.Pointer
var defaultUrl string
var hupChan = make(chan os.Signal, 1)
var sigChan = make(chan os.Signal, 1)
var quitChan = make(chan bool, 1)

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
	status := 302
	if !ok {
		glog.Warningf("no match for %s", r.URL.Path)
		if defaultUrl != "" {
			dest = defaultUrl
		} else {
			dest = ""
			status = 404
		}
	}
	glog.Infof("%s %s %s %d", r.RemoteAddr, r.URL.Path, dest, status)
	if status == 302 {
		http.Redirect(w, r, dest, status)
	} else {
		http.Error(w, "not found", status)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var pPort = flag.Int("port", 80, "listening port")
	var pUrl = flag.String("defaultUrl", "http://google.com", "default URL")
	flag.Parse()
	loadMap()

	var server *http.Server
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		server.Close()
		close(quitChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		signal.Notify(hupChan, syscall.SIGHUP)
		for {
			select {
			case <-quitChan:
				return
			case <-hupChan:
				glog.Infoln("got SIGHUP; reloading urls.txt")
				loadMap()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		defaultUrl = *pUrl
		http.HandleFunc("/", handler)
		glog.Infof("pid %d; listening on port %d", os.Getpid(), *pPort)

		server = &http.Server{
			Addr:         fmt.Sprintf(":%d", *pPort),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		err := server.ListenAndServe()
		if err != nil {
			glog.Fatalln(err)
		}
	}()

	wg.Wait()
	glog.Info("server stopped")
	glog.Flush()
}
