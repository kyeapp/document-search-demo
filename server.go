//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"

	"github.com/blevesearch/bleve/v2"
)

var bindAddr = flag.String("addr", ":8095", "http listen address")
var dataDir = flag.String("dataDir", "data", "data directory")
var staticBleveMappingPath = flag.String("staticBleveMapping", "",
	"optional path to static-bleve-mapping directory for web resources")

func main() {
	flag.Parse()

	// walk the data dir and register index names
	dirEntries, err := ioutil.ReadDir(*dataDir)
	if err != nil {
		log.Fatalf("error reading data dir: %v", err)
	}

	for _, dirInfo := range dirEntries {
		indexPath := *dataDir + string(os.PathSeparator) + dirInfo.Name()

		// skip single files in data dir since a valid index is a directory that
		// contains multiple files
		if !dirInfo.IsDir() {
			log.Printf("not registering %s, skipping", indexPath)
			continue
		}

		i, err := bleve.Open(indexPath)
		if err != nil {
			log.Printf("error opening index %s: %v", indexPath, err)
			panic("no index")
		}
		log.Printf("registered index: %s", dirInfo.Name())
		// set correct name in stats
		i.SetName(dirInfo.Name())
		i.Close()
	}

	// start the HTTP server
	// http.Handle("/", router)
	http.HandleFunc("/search", searchHandler)
	log.Printf("Listening on %v", *bindAddr)
	log.Fatal(http.ListenAndServe(*bindAddr, addCorsHeaders(http.DefaultServeMux)))
}

type SearchRes struct {
	Name string
	Line []string
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// example query: http://localhost:8095/search?i=hpotter.bleve&q=nimbus
	indexPath := r.URL.Query().Get("i")
	searchTerm := r.URL.Query().Get("q")
	searchResults, err := performSearch(indexPath, searchTerm)
	if err != nil {
		return
	}
	// printStruct(searchResults)
	// fmt.Printf("%v", searchResults)

	hitResp := make([]SearchRes, len(searchResults.Hits))

	for i, hit := range searchResults.Hits {
		hitResp[i].Name = hit.ID
		hitResp[i].Line = hit.Fragments["Line"]
	}

	res := struct {
		SearchStat string
		Hits       []SearchRes
	}{
		SearchStat: fmt.Sprintf("%d results (%s)", searchResults.Total, searchResults.Took),
		Hits:       hitResp,
	}

	jsonResponse, err := json.Marshal(res)
	if err != nil {
		log.Printf("JSON marshaling error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func performSearch(indexPath string, searchTerm string) (*bleve.SearchResult, error) {
	log.Printf(`Searching through index "%s" for "%s"`, indexPath, searchTerm)

	path := *dataDir + string(os.PathSeparator) + indexPath
	index, err := bleve.Open(path)
	if err != nil {
		log.Printf("error opening index %s: %v", indexPath, err)
		return nil, err
	}

	indexQuery := bleve.NewMatchQuery(searchTerm)
	searchReq := bleve.NewSearchRequest(indexQuery)
	searchReq.Size = math.MaxInt64
	searchReq.Highlight = bleve.NewHighlight()
	searchResults, err := index.Search(searchReq)
	if err != nil {
		log.Printf("index search error: %v", err)
		return nil, err
	}

	err = index.Close()
	if err != nil {
		log.Printf("close index error: %v", err)
		return nil, err
	}

	return searchResults, nil
}

func addCorsHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			// Handle preflight requests
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func printStruct(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling struct:", err)
		return
	}
	fmt.Println(string(jsonData))
}
