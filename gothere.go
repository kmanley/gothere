package main

import (
	"bufio"
	"flag"
	"fmt"
	_ "github.com/davecgh/go-spew/spew"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
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
		log.Fatal(err)
	}
	defer file.Close()
	urls := make(UrlMap)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || len(line) < 1 {
			continue
		}
		// TODO: warn if duplicate key!
		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			urls[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
		} else {
			log.Printf("skipping malformed line:\n%s", line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
	log.Printf("loaded %d mappings", len(urls))
	// spew.Dump(urls)
	atomic.StorePointer(&urlmap, (unsafe.Pointer)(&urls))
}

// HTTP handler. If the requested URL is in our map,
// we redirect to the target. Otherwise we redirect to
// the default URL if specified.
func handler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	urls := *(*UrlMap)(atomic.LoadPointer(&urlmap))
	dest, ok := urls[strings.ToLower(r.URL.Path)]
	if !ok {
		if defaultUrl != "" {
			http.Redirect(w, r, defaultUrl, 302)
			return
		} else {
			http.Error(w, "not found", 404)
			return
		}
	}
	http.Redirect(w, r, dest, 302)
}

func main() {
	loadMap()
	signal.Notify(sigChan, syscall.SIGHUP)
	go func() {
		for _ = range sigChan {
			log.Println("got SIGHUP; reloading urls.txt")
			loadMap()
		}
	}()

	var pPort = flag.Int("port", 80, "listening port")
	var pUrl = flag.String("defaultUrl", "http://google.com", "default URL")
	flag.Parse()

	defaultUrl = *pUrl
	http.HandleFunc("/", handler)
	log.Printf("pid %d; listening on port %d", os.Getpid(), *pPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *pPort), nil)
	if err != nil {
		log.Fatal(err)
	}
}
