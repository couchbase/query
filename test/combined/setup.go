//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/couchbase/query/logging"
)

var serverPat = regexp.MustCompile("couchbase-server-enterprise_.*-([0-9]*)-linux_amd64.deb$")

// find the desired (or latest) installation binary .deb package in the local directory tree
func findLocalServer(loc string, bld int) (string, error) {
	d, err := os.Open(loc)
	if err != nil {
		logging.Errorf("Failed to open server location (%s): %v", loc, err)
		return "", err
	}
	if fi, err := d.Stat(); err != nil {
		logging.Errorf("Failed to stat server location: %v", err)
		d.Close()
		return "", err
	} else if !fi.IsDir() {
		d.Close()
		//logging.DBG("%v",loc)
		return loc, nil
	}
	mbld := -1
	var mn string
	for {
		ents, err := d.ReadDir(10)
		if err != nil && err != io.EOF {
			d.Close()
			logging.Errorf("Failed to read server location: %v", err)
			return "", err
		}
		for i := range ents {
			if ents[i].IsDir() {
				n, err := findLocalServer(loc+"/"+ents[i].Name(), bld)
				if err != nil {
					if err != os.ErrNotExist {
						return "", err
					}
				} else if bld != -1 {
					return n, nil
				} else {
					if m := serverPat.FindStringSubmatch(n); len(m) == 2 {
						if v, err := strconv.Atoi(m[1]); err == nil {
							if mbld < v {
								mbld = v
								mn = n
							}
						}
					}
				}
			} else if m := serverPat.FindStringSubmatch(ents[i].Name()); len(m) == 2 {
				if v, err := strconv.Atoi(m[1]); err == nil {
					if bld == v {
						d.Close()
						return loc + "/" + ents[i].Name(), nil
					} else if bld == -1 {
						if mbld < v {
							mbld = v
							mn = loc + "/" + ents[i].Name()
						}
					}
				}
			}
		}
		if len(ents) < 10 {
			break
		}
	}
	if mn == "" {
		return "", os.ErrNotExist
	}
	return mn, nil
}

// find the installation binary path for the desired build
// if the starting location is a local file, use findLocalServer
// if the starting location is a binary download, try use the location as is
// if the starting location is an html file, scour the HREF tags for binary download locations or for build-number relative paths
// inwhich case, descend into the link and check it (strictly this is a recursive action)
// This is indended to be pointed to latestbuilds for the version and for it to be able to location the highest build number
// for which there is a suitable binary to download
func findServer(loc string, bld int) (string, error) {
	//logging.DBG("loc=%s, bld=%d", loc, bld)
	u, err := url.Parse(loc)
	if err != nil {
		logging.Errorf("Failed to parse server location: %v", err)
		return "", err
	}
	if u.Scheme == "" {
		// local directory
		return findLocalServer(loc, bld)
	}
	loc = u.String()
	if !strings.HasSuffix(loc, "/") {
		loc += "/"
	}
	if bld != -1 {
		loc += fmt.Sprintf("%d/", bld)
	}
	var refs []string
	refs, err = getRefs(loc, refs)
	if err != nil {
		return "", err
	}
	var blds []int
	for i := 0; i < len(refs); i++ {
		if strings.HasSuffix(refs[i], "/") {
			n := strings.LastIndex(refs[i][:len(refs[i])-1], "/") + 1
			if v, err := strconv.Atoi(refs[i][n : len(refs[i])-1]); err == nil {
				if bld == v {
					refs, err = getRefs(refs[i], refs)
					if err != nil {
						return "", err
					}
				} else if bld == -1 {
					blds = append(blds, v)
				}
			}
		} else if m := serverPat.FindStringSubmatch(refs[i]); len(m) == 2 {
			if v, err := strconv.Atoi(m[1]); err == nil {
				if bld == -1 && len(refs) > 1 {
					blds = append(blds, v)
				} else if v == bld || len(refs) == 1 {
					return refs[i], nil
				}
			}
		}
	}
	if len(blds) == 0 {
		logging.Errorf("Failed to find server package for build: %v", bld)
		return "", os.ErrNotExist
	}
	sort.Ints(blds)
	for len(blds) > 0 {
		refs, err = getRefs(loc+fmt.Sprintf("%d/", blds[len(blds)-1]), nil)
		for i := range refs {
			if m := serverPat.FindStringSubmatch(refs[i]); len(m) == 2 {
				return refs[i], nil
			}
		}
		blds = blds[:len(blds)-1]
	}
	return "", os.ErrNotExist
}

// interrogates a URL: if text/html, gathers contained HREF targets (as relative locations)
// if not text/html, returns the location (without a trailing path separator if one is present)
func getRefs(loc string, refs []string) ([]string, error) {
	//logging.DBG("%v",loc)
	resp, err := http.Get(loc)
	if err != nil {
		logging.Errorf("Failed to read location (%s): %v", loc, err)
		return refs, err
	}
	isHtml := false
	if v, ok := resp.Header["Content-Type"]; ok {
		for i := range v {
			if v[i] == "text/html" {
				isHtml = true
				break
			}
		}
	}
	if !isHtml {
		//logging.DBG("Not html")
		resp.Body.Close()
		refs = append(refs, strings.TrimSuffix(loc, "/"))
		return refs, nil
	}
	defer resp.Body.Close()
	tok := html.NewTokenizer(resp.Body)
	for {
		tt := tok.Next()
		switch {
		case tt == html.ErrorToken:
			return refs, nil
		case tt == html.StartTagToken:
			t := tok.Token()
			for _, a := range t.Attr {
				if a.Key == "href" {
					u, err := url.Parse(a.Val)
					if err == nil && u.Scheme == "" {
						refs = append(refs, loc+a.Val)
					}
				}
			}
		}
	}
}

// dowloads the URI to a local file in the temporary directory with the same name as the final portion of the target
// if the local file already exists and the size of the URI target and file match, then the download is skipped
// the path to the local file is returned
func download(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		logging.Errorf("Failed to parse server product binary location: %v", err)
		return "", err
	}
	if u.Scheme == "" || u.Scheme == "file" {
		return u.RequestURI(), nil
	}
	target := path.Join(os.TempDir(), path.Base(u.RequestURI()))
	f, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logging.Errorf("Failed to create local file \"%s\" for download: %v", target, err)
		return "", err
	}
	stat, err := f.Stat()
	if err != nil {
		logging.Errorf("Failed to stat local file \"%s\": %v", f.Name(), err)
		f.Close()
		return "", err
	}
	resp, err := http.Get(uri)
	if err != nil {
		logging.Errorf("Failed to download server binary (%s): %v", uri, err)
		return "", err
	}
	defer resp.Body.Close()
	if stat.Size() > 0 {
		if v, ok := resp.Header["Content-Length"]; ok {
			if l, err := strconv.Atoi(v[0]); err == nil && int64(l) == stat.Size() {
				logging.Infof("Local file size matches; skipping download. Using: %s", f.Name())
				f.Close()
				return f.Name(), nil
			}
		}
	}
	logging.Infof("Downloading \"%s\" to \"%s\"...", path.Base(u.RequestURI()), f.Name())
	n, err := io.Copy(f, resp.Body)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		logging.Errorf("Failed to download \"%s\" to \"%s\" (%v bytes copied): %v", uri, f.Name(), n, err)
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

// installs the package pointed to using the Debian package manager (this is intended for Debian based machines only)
// first it attempts to remove the previously installed version, if any.  This is to avoid upgrade/downgrade issues.
func install(pkg string) error {
	logging.Debugf("%s", pkg)
	dpkg, err := exec.LookPath("dpkg")
	if err != nil {
		logging.Errorf("Failed to locate dpkg in the PATH: %v", err)
		return err
	}

	// backup the query.log file before removing
	if src, err := os.Open("/opt/couchbase/var/lib/couchbase/logs/query.log"); err == nil {
		if dst, err := os.CreateTemp(os.TempDir(), "query.log_zip_*"); err == nil {
			zip := gzip.NewWriter(dst)
			if _, err := io.Copy(zip, src); err == nil {
				logging.Infof("query.log backed up to %s", dst.Name())
			} else {
				logging.Errorf("Failed to back up query.log: %v", err)
			}
		}
	}

	logging.Debugf("removing")
	rm := exec.Command(dpkg, "--no-pager", "-P", "couchbase-server")
	err = rm.Run()
	if err != nil {
		logging.Debugf("Package removal returned: %v", err)
	}

	// ensure the installation location is clear
	logging.Debugf("cleaning up")
	keep := []string{".keep"}
	// if it exists, the .keep file lists any /opt/couchbase directory entries to not remove
	content, err := os.ReadFile("/opt/couchbase/.keep")
	if err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			n := strings.TrimSpace(line)
			if len(n) > 0 {
				keep = append(keep, n)
			}
		}
	}
	dir, err := os.ReadDir("/opt/couchbase")
	if err == nil {
		for _, d := range dir {
			keepEntry := false
			for i := range keep {
				if d.Name() == keep[i] {
					keepEntry = true
					break
				}
			}
			if !keepEntry {
				rm = exec.Command("rm", "-rf", "/opt/couchbase/"+d.Name())
				//logging.DBG("%v", rm.String())
				err = rm.Run()
				if err != nil {
					logging.Debugf("%v: %v", rm.String(), err)
				}
			}
		}
	}

	logging.Debugf("installing")
	install := exec.Command(dpkg, "-i", pkg)
	stdout, err := install.StdoutPipe()
	if err != nil {
		logging.Errorf("Error setting up installation process: %v", err)
		return err
	}
	stderr, err := install.StderrPipe()
	if err != nil {
		logging.Errorf("Error setting up installation process: %v", err)
		return err
	}
	if err = install.Start(); err != nil {
		logging.Errorf("Error installing '%s': %v", pkg, err)
		return err
	}
	var stdo strings.Builder
	var stde strings.Builder
	io.Copy(&stdo, stdout)
	io.Copy(&stde, stderr)
	if err = install.Wait(); err != nil {
		logging.Errorf("Error installing '%s': %v", pkg, err)
		for _, s := range strings.Split(stdo.String(), "\n") {
			logging.Errorf("STDOUT: %s", s)
		}
		for _, s := range strings.Split(stde.String(), "\n") {
			logging.Errorf("STDERR: %s", s)
		}
		return err
	}
	return nil
}

// returns the build number of the installed build, or -1 if there isn't one or it can't be determined
func installedBuild() int {
	f, err := os.Open("/opt/couchbase/VERSION.txt")
	if err != nil {
		return -1
	}
	b, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return -1
	}
	n := bytes.IndexByte(b, '-')
	if n == -1 {
		return -1
	}
	i, err := strconv.Atoi(string(bytes.TrimSuffix(b[n+1:], []byte{'\n'})))
	if err != nil {
		return -1
	}
	return i
}

// checks the build number in the given URI is the supplied build
func isSameBuild(uri string, bld int) bool {
	var v int
	var err error
	if m := serverPat.FindStringSubmatch(uri); len(m) == 2 {
		if v, err = strconv.Atoi(m[1]); err == nil {
			return v == bld
		}
	}
	return false
}

// checks to see if the currently installed build can and should be updated
// does nothing if the process isn't running as root
func installServer(c map[string]interface{}, force bool) error {
	cs, ok := c["cluster_setup"].(map[string]interface{})
	if !ok {
		logging.Errorf("Configuration element \"cluster_setup\" not found or not an object.")
		return os.ErrNotExist
	}

	ibld := installedBuild()
	logging.Debugf("Installed build: %v", ibld)
	if uid := os.Getuid(); uid != 0 {
		if ibld == -1 {
			logging.Infof("No installed build and installation not possible as uid: %v", uid)
			return os.ErrNotExist
		}
		logging.Infof("Running as uid %d.  Using existing installed build: %d", uid, ibld)
		return nil
	}
	bld := -1
	if f, ok := cs["build"].(float64); ok {
		bld = int(f)
	} else if s, ok := cs["build"].(string); ok {
		if v, err := strconv.Atoi(s); err == nil {
			bld = v
		}
	}
	if bld == ibld && bld != -1 && !force {
		logging.Infof("Build (%d) already installed.", ibld)
		return nil
	}
	loc, _ := cs["server_location"].(string)
	logging.Debugf("Looking for build: %v in %s", bld, loc)
	if u, err := findServer(loc, bld); err != nil {
		return err
	} else {
		if !isSameBuild(u, ibld) || force {
			if f, err := download(u); err != nil {
				return err
			} else {
				if err = install(f); err != nil {
					return err
				}
			}
		} else {
			logging.Infof("Build (%d) already installed.", ibld)
		}
	}
	return nil
}

// creates the couchbase server instance and specified keyspaces
func configureInstance(c map[string]interface{}) error {
	//logging.Debugf("%v", c)
	var initArgs string
	var ok bool
	var err error

	cs, ok := c["cluster_setup"].(map[string]interface{})
	if !ok {
		logging.Fatalf("Configuration element \"cluster_setup\" not found or not an object.")
		return os.ErrNotExist
	}

	if initArgs, ok = cs["cluster-init"].(string); !ok {
		initArgs = "-c localhost --cluster-username Administrator --cluster-password password --services query,data,index " +
			"--cluster-ramsize 8192 --cluster-index-ramsize 512"
	}
	base := []string{"cluster-init"}
	args := append(base, strings.Split(initArgs, " ")...)
	logging.Infof("cluster-init args: %v", args)

	if !checkWait(_NODE_URL, "Waiting for cluster manager prior to creating cluster...") {
		return fmt.Errorf("Unable to configure instance.")
	}

	for retry := 1; retry <= _INSTANCE_RETRY_COUNT; retry++ {
		logging.Infof("Attempting to create cluster (%d/%d).", retry, _INSTANCE_RETRY_COUNT)

		ic := exec.Command("/opt/couchbase/bin/couchbase-cli", args...)
		sb := &strings.Builder{}
		ic.Stdout = sb
		err = ic.Run()
		logging.Infof("Server creation response: %v", strings.TrimSuffix(sb.String(), "\n"))
		if err != nil {
			err = fmt.Errorf("%v: %s", err, strings.TrimSuffix(sb.String(), "\n"))
			if !isHttpConnError(err) {
				break
			}
		} else {
			break
		}
		time.Sleep(_RETRY_WAIT)
	}
	waitMigration := false
	if err != nil {
		if strings.Contains(err.Error(), "Cluster is already initialized") {
			logging.Infof("Cluster already initialised; using existing cluster.")
			err = nil
			if rs, ok := cs["restart"].(bool); ok && rs {
				if err = restartCluster(); err != nil {
					logging.Errorf("Cluster restart failed: %v", err)
				}
				waitMigration = true
			}
		} else {
			logging.Errorf("Failed to initialise instance: %v", err)
		}
	} else {
		logging.Infof("Cluster initialised.")
		waitMigration = true
	}

	// wait for the instance to be available
	u, _ := url.JoinPath(_QUERY_URL, "/admin/ping")
	if !checkWait(u, "Waiting for query service to become available...") {
		return fmt.Errorf("Query service did not start in time.")
	}
	if waitMigration {
		logging.Infof("Waiting for Query migration checks.")
		time.Sleep(_MIGRATION_WAIT_TIME)
	}
	return err
}

func restartCluster() error {
	logging.Infof("Attempting to restart cluster.")

	ic := exec.Command("/usr/bin/systemctl", "restart", "couchbase-server")
	sb := &strings.Builder{}
	ic.Stdout = sb
	err := ic.Run()
	logging.Debugf("Server restart response: %v", strings.TrimSuffix(sb.String(), "\n"))
	if err == nil {
		if !checkWait(_NODE_URL, "Waiting for cluster to restart ...") {
			err = fmt.Errorf("Cluster did not restart in time.")
		}
	}
	return err
}
