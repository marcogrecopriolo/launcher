// Copyright 2020 Marco Greco marcogrecopriolo@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	_CONFIG = "launcher.json"

	_DEF_PORT         = ":8084"
	_RESOURCES        = "resources"
	_SETTINGS         = "settings"
	_API              = "api"
	_API_START        = "on"
	_API_STOP         = "off"
	_AUTHENTICATE_MSG = "Credentials fpr remote launcher"

	_DEF_TITLE    = "remote launcher at"
	_ON_IMG       = "on.png"
	_OFF_IMG      = "off.png"
	_SETTINGS_IMG = "refresh.png"
	_ON_ALT       = "is running"
	_OFF_ALT      = "is not runng"
	_SETTINGS_ALT = "refresh"
)

type launcher struct {
	config config
}

type config struct {
	Port     string         `json:"port"`
	User     string         `json:"user"`
	Password string         `json:"password"`
	Title    string         `json:"title"`
	Apps     map[string]App `json:"apps"`
}

type App struct {
	Start  Cmd `json:"start"`
	Status Cmd `json:"status"`
}

type Cmd struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

func main() {
	var l launcher
	var apps []string

	// parse json config
	parseConfig := func() error {
		var conf config

		jsonFile, err := os.Open(_CONFIG)
		if err != nil {
			return err
		}
		byteValue, _ := ioutil.ReadAll(jsonFile)
		jsonFile.Close()

		// det defaults
		conf.Port = _DEF_PORT
		n, _ := os.Hostname()
		conf.Title = _DEF_TITLE + " " + n
		err = json.Unmarshal(byteValue, &conf)
		if err != nil {
			return err
		}
		if len(conf.Apps) == 0 {
			return fmt.Errorf("no apps found")
		}

		l.config = conf

		// sort the names, get round maps random range
		apps = apps[:0]
		for k := range l.config.Apps {
			apps = append(apps, k)
		}
		sort.Strings(apps)
		return nil
	}
	err := parseConfig()
	if err != nil {
		fmt.Printf("gotcha%v\n", err)
		return
	}

	// start listener
	http.Handle("/favicon.ico", http.FileServer(http.Dir("./"+_RESOURCES)))
	http.Handle("/"+_RESOURCES+"/", http.FileServer(http.Dir("./")))
	http.HandleFunc("/"+_API, func(w http.ResponseWriter, r *http.Request) {
		if l.config.User != "" {
			user, passwd, ok := r.BasicAuth()
			if !ok || user != l.config.User || passwd != l.config.Password {
				w.Header()["WWW-Authenticate"] = []string{"Basic realm=" + _AUTHENTICATE_MSG}
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintf(w, "unauthorized")
				return
			}
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "method not allowed")
			return
		}
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "error processing request: %v", err)
			return
		}
		for k, v := range r.Form {
			if _, ok := l.config.Apps[k]; !ok {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "error processing request: %v is not a known app", k)
				return
			}
			if len(v) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "error processing request: too many or too few actions for %v", k)
				return
			}

			if v[0] != _API_START && v[0] != _API_STOP {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "error processing request: %v is and invalid action for %v", v[0], k)
				return
			}
		}
		for k, v := range r.Form {
			var err error

			app := l.config.Apps[k]
			if v[0] == _API_START {
				err = app.start()
			} else {
				err = app.stop()
			}
			if err != nil {
				fmt.Fprintf(w, "error processing request: %v returned %v", k, err)
				return
			}
		}

		// wait a little before refreshing
		time.Sleep(200 * time.Millisecond)
		listApps(l, apps, w, r)
	})

	http.HandleFunc("/"+_SETTINGS, func(w http.ResponseWriter, r *http.Request) {
		err := parseConfig()
		if err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
		listApps(l, apps, w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		listApps(l, apps, w, r)
	})

	http.ListenAndServe(l.config.Port, nil)
}

func listApps(l launcher, apps []string, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<HTML>\n<HEAD>\n<TITLE>%v</TITLE>\n<LINK rel=\"stylesheet\"  href=\"resources/launcher.css\">\n</HEAD>\n", l.config.Title)
	fmt.Fprintf(w, "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<BODY>\n")
	fmt.Fprintf(w, "<DIV id=\"header\">\n<H1>%v</H1>\n</DIV>\n", l.config.Title)
	fmt.Fprintf(w, "<DIV id=\"settings\"><A HREF=%v><IMG class=\"settings\" src=\"%v/%v\" alt=\"%v\"></A></DIV>\n", _SETTINGS, _RESOURCES, _SETTINGS_IMG, _SETTINGS_ALT)
	fmt.Fprintf(w, "<DIV id=\"main\">\n")
	for _, k := range apps {
		var img, alt, api string

		v := l.config.Apps[k]
		if v.status() {
			img = _ON_IMG
			alt = k + " " + _ON_ALT
			api = _API_STOP
		} else {
			img = _OFF_IMG
			alt = k + " " + _OFF_ALT
			api = _API_START
		}
		fmt.Fprintf(w, "<DIV id=\"content\"><A HREF=%v?%v=%v><IMG class=\"content\" src=\"%v/%v\" alt=\"%v\"> %v</A></DIV>\n", _API, k, api, _RESOURCES, img, alt, k)
	}
	fmt.Fprintf(w, "</DIV>\n</BODY>\n</HTML>\n")
}

func (this *App) start() error {

	// don't run twice
	pid, _ := this.pid()
	if pid > 0 {
		return nil
	}

	// FIXME save child on exit
	cmd := exec.Command(this.Start.Cmd, this.Start.Args...)
	return cmd.Run()
}

func (this *App) stop() error {

	// ignotr pid error on stop
	pid, _ := this.pid()

	// only stop if found, and it's not init
	if pid > 1 {
		p, _ := os.FindProcess(pid)
		if p != nil {
			p.Signal(syscall.SIGTERM)
		}
	}
	return nil
}

func (this *App) status() bool {
	pid, _ := this.pid()
	return pid > 0
}

func (this *App) pid() (int, error) {
	cmd := exec.Command(this.Status.Cmd, this.Status.Args...)
	out, err := cmd.Output()
	if err != nil {
		return -1, err
	}
	txt := strings.SplitN(string(out), "\n", 2)
	pid, err := strconv.Atoi(txt[0])
	if err != nil || pid < 1 {
		return -1, err
	}
	return pid, nil
}
