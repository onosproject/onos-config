// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

/*
Package restconf partially implements a Restconf interface for onos-config
*/
package restconf

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/opennetworkinglab/onos-config/pkg/listener"
	"github.com/opennetworkinglab/onos-config/pkg/store"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const topDir = "restconf"

const (
	mimeAnything    = "*/*"
	mimeJSON        = "application/json"
	mimeYangJSON    = "application/yang-data+json"
	mimePlain       = "text/plain"
	mimeXML         = "application/xml"
	mimeYangXML     = "application/yang-data+xml"
	mimeEventStream = "text/event-stream"
)

const hostMeta = "<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>\n" +
	"<Link rel='restconf' href='{{.}}'/>\n" +
	"</XRD>\n"

var (
	configStore *store.ConfigurationStore
	changeStore *store.ChangeStore
)

// StartRestServer is a go routine function that runs the restconf server
func StartRestServer(port int,
	config *store.ConfigurationStore,
	change *store.ChangeStore) error {

	configStore = config
	changeStore = change

	http.HandleFunc("/.well-known/host-meta", hostMetaHandler)
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/restconf/", restconfHandler)
	http.HandleFunc("/restconf/change/", changeHandler)
	http.HandleFunc("/restconf/configuration/", configurationHandler)
	http.HandleFunc("/restconf/data/", treeHandler)
	http.HandleFunc("/restconf/events/", streamsHandler)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	log.Fatal(err)
	return err
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/restconf/", http.StatusFound)
}

func restconfHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Content-Type", mimeJSON)
		options := make(map[string][]string)
		options["change"] = []string{"GET", "PUT", "POST"}
		options["configuration"] = []string{"GET", "PUT", "POST"}
		options["data"] = []string{"GET"}
		jsonEncoder := json.NewEncoder(w)
		jsonEncoder.Encode(options)
	}
	msg := fmt.Sprintf("Method %s not allowed on %s", r.Method, topDir)
	http.Error(w, msg, http.StatusMethodNotAllowed)
}

func treeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		var isFirst = true
		fmt.Fprintf(w, "{\"Devices\": [")
		for _, conf := range configStore.Store {
			treeBytes, err := BuildTree(conf.ExtractFullConfig(changeStore.Store, 0))
			if err != nil {
				http.Error(w, err.Error(), http.StatusMethodNotAllowed)
				return
			}
			if isFirst {
				isFirst = false
			} else {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, "{\"Device\": \"%s\", \"data\":", conf.Device)
			fmt.Fprintln(w, string(treeBytes), "}")
		}
		fmt.Fprintf(w, "]}")
	}
}

// streamsHandler is for supporting Server Sent Events
// https://www.w3.org/TR/eventsource/
func streamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.Header.Get("Accept") != mimeEventStream {
		http.Error(w, "Only GET with 'Accept=text/event-stream' are allowed",
			http.StatusMethodNotAllowed)
		return
	}
	// Make sure the writer supports flushing of buffer
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get a hash of the request - uniquely identify it
	h := sha1.New()
	requestID := fmt.Sprintf("%s%s%v", r.Host, r.RemoteAddr, time.Now())
	_, err := io.WriteString(h, requestID)
	if err != nil {
		http.Error(w, "Unable to make hash from request!", http.StatusInternalServerError)
		return
	}

	channelIdentifier := store.B64(h.Sum(nil))
	nbiChan, err1 := listener.Register(channelIdentifier, false)
	if err1 != nil {
		http.Error(w, "Unable to subscribe to events for "+channelIdentifier,
			http.StatusInternalServerError)
		return
	}

	// Listen to connection close and un-register messageChan
	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		listener.Unregister(channelIdentifier, false)
		log.Println("Unregistered listener for client", channelIdentifier)
	}()

	// block waiting for messages broadcast on this connection's nbiChan
	log.Println("Ready for new events for client", channelIdentifier)
	i := 0
	for nbiChange := range nbiChan {
		// Write to the ResponseWriter
		// Server Sent Events compatible
		fmt.Fprintf(w, "id: %d\n", i)
		fmt.Fprintf(w, "event: %s\n", nbiChange.EventType())
		fmt.Fprintf(w, "time: %s\n", nbiChange.Time().Format(time.RFC3339))
		valuesjson, _ := json.Marshal(nbiChange.Values())
		fmt.Fprintf(w, "data: %s %s\n\n",
			nbiChange.Subject(), valuesjson)

		// Flush the data immediatly instead of buffering it for later.
		flusher.Flush()
		i++
	}
	log.Println("Finished with", channelIdentifier)
}

func hostMetaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/xrd+xml")
		t, err := template.New("hostmeta.xml").Parse(hostMeta)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = t.Execute(w, topDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func changeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		switch r.Header.Get("Accept") {
		case mimeXML, mimeYangXML:
			http.Error(w, "<XML Encoding not supported/>", http.StatusNotImplemented)
			return
		case mimeYangJSON:
			w.Header().Set("Content-Type", mimeYangJSON)
		case mimeAnything, mimePlain, mimeJSON:
			w.Header().Set("Content-Type", mimeJSON)
		default:
			msg := fmt.Sprintf("Unexpected mimetype %s", r.Header.Get("Accept"))
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		jsonEncoder := json.NewEncoder(w)
		jsonEncoder.Encode(changeStore.Store)
	}
}

func configurationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		jsonEncoder := json.NewEncoder(w)
		jsonEncoder.Encode(configStore.Store)
	}
}
