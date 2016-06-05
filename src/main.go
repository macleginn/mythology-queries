package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
)

// The program consists of an a http server that listens on port ...
// and accepts requests of the following four kinds:
// (1) .../motifQuery&code=...&num=...
// (2) .../traditionQuery&code=...&num=...
// (3) .../fetchMotifDict
// (4) .../fetchTraditionDict
// To queries of the type (1) or (2) the server returns N=num motifs
// closest in the spatial distribution to the given motif or N=num
// traditions closest in the inventory of motifs to the given tradition
// formatted as JSON:
// {
// 	"motif/tradition-code": int,
// 	"closest-motifs/traditions": [
// 		motif/tradition-code-1,
// 		motif/tradition-code-2,
// 		...
// 	]
// }
// fetchTraditionDict and fetchMotifDict return the description
// of motifs and traditions necessary for placing the them on the map
// displaying the results.

// The basic distance function used by both query handlers.
func manhattan(v1 []int, v2 []int) (int, error) {
	if len(v1) != len(v2) {
		return -1, errors.New("The vectors must be of the same length")
	}
	distance := 0
	for i := range v1 {
		diff := v1[i] - v2[i]
		if diff < 0 {
			distance += -diff
		} else {
			distance += diff
		}
	}
	return distance, nil
}

// The same type of handler is used both for motif and tradition
// queries since they do the same thing: compute Manhattan
// distances between a given item and all other items in the collection.
// The difference lies in the representation that is used to
// initialise the distance comparator.
type queryHandler struct {
	items    map[string]bool
	distance func(mCode1, mCode2 string) (int, error)
}

// Initialise comparator with a closure.
func initialiseComparator(representations map[string][]int) func(item1, item2 string) (int, error) {
	return func(item1, item2 string) (int, error) {
		v1 := representations[item1]
		v2 := representations[item2]
		return manhattan(v1, v2)
	}
}

// A type satisfying sort.Interface for returning n closest motifs/traditions.
type neighbour struct {
	code     string
	distance int
}

type neighbours []neighbour

func (n neighbours) Len() int { return len(n) }

func (n neighbours) Swap(i, j int) { n[i], n[j] = n[j], n[i] }

func (n neighbours) Less(i, j int) bool {
	return n[i].distance < n[j].distance
}

// The workhorse function.
func (q *queryHandler) getNNearestNeighbours(item string, n int) neighbours {
	allNeighbours := make(neighbours, 0)
	for storageItem, _ := range q.items {
		if storageItem == item {
			continue
		}
		distance, err := q.distance(item, storageItem)
		if err != nil {
			log.Fatal(err)
		}
		allNeighbours = append(allNeighbours, neighbour{
			storageItem,
			distance,
		})
	}
	sort.Sort(allNeighbours)
	if n > len(allNeighbours) || n == -1 {
		return allNeighbours
	} else {
		return allNeighbours[:n]
	}
}

func (q *queryHandler) handleQuery(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	code, ok := query["code"]
	// Check that a code is present and that there is only of them.
	if !ok || len(code) != 1 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if _, ok = q.items[code[0]]; !ok {
		http.Error(w, "Wrong motif/tradition code", http.StatusNotFound)
		return
	}
	// Check if a number of requested items is present and set the number to -1
	// if it is not or if there are several.
	// TODO: come up with a reasonable default number of items to return.
	ntrads := -1
	var err error
	if n, ok := query["n"]; ok && len(n) == 1 {
		ntrads, err = strconv.Atoi(n[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	data := make([]map[string]string, 0)
	for _, val := range q.getNNearestNeighbours(code[0], ntrads) {
		data = append(data, map[string]string{
			"code":     val.code,
			"distance": strconv.Itoa(val.distance)})
	}
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Fatal(err)
	}

	// Send data
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(dataJSON)
}

// Closure for fetching descriptions of motifs and traditions.
func createJSONServer(data []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "Bad request: %s", r.URL.Path[1:])
}

func main() {
	// Prepare and deliver traditionDict
	// TODO
	traditionDict := map[string]string{
		"1": "English",
		"2": "Sinai Bedouins",
		"3": "French",
	}
	traditionDictJSON, err := json.MarshalIndent(traditionDict, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/fetchTraditionDict", createJSONServer(traditionDictJSON))

	// Prepare and deliver motifDict
	// TODO
	motifDict := map[string]string{
		"a1":   "Killing the sun",
		"b13":  "Dying for love",
		"l109": "An ogre is vindicated",
	}
	motifDictJSON, err := json.MarshalIndent(motifDict, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/fetchMotifDict", createJSONServer(motifDictJSON))

	// Initialise data for motifHandler
	// TODO
	motifRepresentations := map[string][]int{
		"a1":   []int{1, 0, 1, 1},
		"b13":  []int{1, 1, 0, 0},
		"l109": []int{1, 1, 1, 1},
	}

	// Initialise motifHandler with the data and register it with the server
	motifHandler := queryHandler{}
	motifHandler.distance = initialiseComparator(motifRepresentations)
	motifHandler.items = make(map[string]bool)
	for key, _ := range motifRepresentations {
		motifHandler.items[key] = true
	}

	http.HandleFunc("/motifQuery", motifHandler.handleQuery)

	// Initialise data for traditionHandler
	// TODO
	traditionRepresentations := map[string][]int{
		"1": []int{1, 1, 1},
		"2": []int{0, 1, 0},
		"3": []int{1, 0, 1},
	}

	// Initialise traditionHandler with the data and register it with the server
	traditionHandler := queryHandler{}
	traditionHandler.distance = initialiseComparator(traditionRepresentations)
	traditionHandler.items = make(map[string]bool)
	for key, _ := range traditionRepresentations {
		traditionHandler.items[key] = true
	}

	http.HandleFunc("/traditionQuery", traditionHandler.handleQuery)

	// Requests for all other paths are bad by definition
	http.HandleFunc("/", errorHandler)
	http.ListenAndServe(":8080", nil)
}
