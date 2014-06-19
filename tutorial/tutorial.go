package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/couchbaselabs/clog"
	"github.com/russross/blackfriday"
)

type Index map[string]int

func setup(cacheDir string, src string, srcF os.FileInfo, idx Index, err error) error {
	if strings.HasPrefix(srcF.Name(), ".") || strings.Contains(src, "/.") {
		return nil
	}

	dst := cacheDir + src
	if strings.HasSuffix(dst, ".md") {
		dst = strings.TrimSuffix(dst, ".md") + ".html"
	}
	dstF, dstE := os.Stat(dst)

	// if up to date, skip
	if !os.IsNotExist(dstE) &&
		(srcF.IsDir() || dstF.ModTime().After(srcF.ModTime())) {
		return nil
	}

	// copy to cache
	if os.IsNotExist(dstE) {
		clog.Log("Copying %s", srcF.Name())
	} else {
		clog.Log("Updating %s", srcF.Name())
	}

	if srcF.IsDir() {
		if err := os.MkdirAll(dst, srcF.Mode()); err != nil {
			return err
		}
		return nil
	}

	sbuf, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	dbuf := sbuf
	if strings.HasSuffix(src, ".md") {
		dbuf = blackfriday.MarkdownCommon(sbuf)
	}
	if err := ioutil.WriteFile(dst, dbuf, srcF.Mode()); err != nil {
		return err
	}

	if strings.HasSuffix(src, ".md") {
		search := regexp.MustCompile("##[^\n]+\n")
		found := search.FindString(string(sbuf))
		if len(found) > 0 {
			found = strings.TrimSpace(strings.TrimPrefix(found, "##"))
			nsearch := regexp.MustCompile("/slide-(\\d+).md")
			num := nsearch.FindStringSubmatch(src)
			if num != nil {
				n, _ := strconv.Atoi(num[1])
				li := strconv.Itoa(n) + ". " + found
				idx[li] = n
			}
		}
	}

	return nil
}

func main() {
	xsrc := flag.String("src", "./content", "Source directory to read markdown from")
	xdst := flag.String("dst", "", "Destination to write translated content")
	flag.Parse()

	srcD, srcE := os.Stat(*xsrc)
	if os.IsNotExist(srcE) || !srcD.IsDir() {
		clog.Fatalf("Source directory does not exist: %s", *xsrc)
	}

	if len(*xdst) > 0 {
		dstD, dstE := os.Stat(*xdst)
		if os.IsNotExist(dstE) || !dstD.IsDir() {
			clog.Fatalf("Target directory does not exist: %s", *xdst)
		}
	}

	tld := strings.Trim(*xsrc, "/")
	dirpos := strings.LastIndex(tld, "/")
	if dirpos < 0 {
		tld = tld[dirpos+1:]
	}
	if len(tld) < 1 {
		clog.Fatalf("Source directory path must be at least one level deep, example: ./content")
	}

	if len(*xdst) > 0 {
		translate(*xsrc, *xdst, tld)
	} else {
		serve(*xsrc, tld)
	}
}

func isEmpty(dir string) bool {
	empty := true
	walker := func(dir string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		empty = false
		return nil
	}
	if err := filepath.Walk(dir, walker); err != nil {
		clog.Fatalf("Filewalk %v", err)
	}
	return empty
}

func translate(src, dst, tld string) {

	if !isEmpty(dst) {
		clog.Fatalf("Target directory %s is not empty", dst)
	}

	var idx Index = make(map[string]int)
	walker := func(fsrc string, f os.FileInfo, err error) error {
		return setup(dst, fsrc, f, idx, err)
	}
	if err := filepath.Walk(src, walker); err != nil {
		clog.Fatalf("Filewalk %v", err)
	}

	json, err := json.Marshal(idx)
	if err != nil {
		clog.Fatalf("During Index JSON Marshal %v", err)
	}

	jfile := strings.TrimRight(dst, "/")
	jfile += "/" + tld + "/index.json"
	err = ioutil.WriteFile(jfile, json, 0666)
	if err != nil {
		clog.Fatalf("Error writing json file: %s", jfile)
	}
}

func serve(cdir string, tld string) {
	tempDir, _ := ioutil.TempDir("", "tut")
	tempDir += string(os.PathSeparator)
	defer os.RemoveAll(tempDir)
	clog.Log("Workdir %s", tempDir)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		os.RemoveAll(tempDir)
		clog.Fatal("Stopped")
	}()

	var idx Index
	idx = make(map[string]int)

	walker := func(src string, f os.FileInfo, err error) error {
		return setup(tempDir, src, f, idx, err)
	}

	if err := filepath.Walk(cdir, walker); err != nil {
		clog.Fatalf("Filewalk %v", err)
	}

	getindex := func(w http.ResponseWriter, r *http.Request) {
		json, err := json.Marshal(idx)
		if err != nil {
			clog.Fatalf("During Index JSON Marshal %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
	http.HandleFunc("/tutorial/index.json", getindex)

	url, _ := url.Parse("http://localhost:8093")
	rp := httputil.NewSingleHostReverseProxy(url)
	http.Handle("/query", rp)

	fs := http.FileServer(http.Dir(tempDir + "/" + tld + "/"))
	http.Handle("/tutorial/", http.StripPrefix("/tutorial/", fs))

	http.Handle("/", http.RedirectHandler("/tutorial/index.html#1", 302))

	clog.Log("Running at http://localhost:8000/")
	go func() {
		for {
			filepath.Walk(cdir, walker)
			time.Sleep(2 * time.Second)
		}
	}()

	// last step
	if err := http.ListenAndServe(":8000", nil); err != nil {
		clog.Fatalf("ListenAndServe %v", err)
	}
}
