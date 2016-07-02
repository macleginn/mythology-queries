package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/golang/geo/s2"
)

// The program consists of an a http server that listens on port ...
// and accepts requests of the following kinds:
// (1) .../motifQuery?code=...&num=...
// (2) .../traditionQuery?code=...&num=...
// (3) .../fetchTraditionDict
// (4) .../fetchMotifDistr?code=...
// (5) .../fetchMotifList
// (6) .../compareTraditions?trad1=...&trad2=...
// To queries of the type (1) or (2) the server returns N=num motifs
// closest in the spatial distribution to the given motif or N=num
// traditions closest in the inventory of motifs to the given tradition
// formatted as JSON:
// [
//   {
//     "code": "Nyangi",
//     "distance": "2"
//   },
//   {
//     "code": "Turkana, Toposa",
//     "distance": "6"
//   }
// ]
// (for traditions) or:
// [
//   {
//     "code": "b24a_7",
//     "distance": "1"
//   },
//   {
//     "code": "i28a_3",
//     "distance": "3"
//   }
// ]
// (for motifs).
// fetchTraditionDict returns the description of traditions necessary
// for placing the them on the map and displaying the results.
// fetchMotifDistr returns the vector of traditions having the motif
// for showing its distribution on the map.
// fetchMotifList returns a list of motifs in the database.
// compareTraditions returns three lists:
// * a list of motifs common to both traditions
// * a list of motifs present only in the first tradition
// * a list of motifs present only in the second tradition

// The basic distance function used by both query handlers.
func manhattan(v1 []int, v2 []int) (float64, error) {
	if len(v1) != len(v2) {
		return -1.0, errors.New("Vectors must be of the same length")
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
	return float64(distance), nil
}

func hav(θ float64) float64 {
	return (1 - math.Cos(θ)) / 2
}

func havDist(lat1, lon1, lat2, lon2 float64) float64 {
	R := 6371.0
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	λ1 := lon1 * math.Pi / 180
	λ2 := lon2 * math.Pi / 180
	return 2 * R * math.Asin(math.Sqrt(hav(φ2-φ1)+math.Cos(φ1)*math.Cos(φ2)*hav(λ2-λ1)))
}

// The same type of handler is used both for motif and tradition
// queries since they do the same thing: compute a bunch of
// distances between a given item and all other items in the collection.
type queryHandler struct {
	items    map[string]bool
	distance func(mCode1, mCode2 string) (float64, error)
}

// A type satisfying sort.Interface for returning n closest motifs/traditions.
type neighbour struct {
	code     string
	distance float64
}

type neighbours []neighbour

func (n neighbours) Len() int { return len(n) }

func (n neighbours) Swap(i, j int) { n[i], n[j] = n[j], n[i] }

func (n neighbours) Less(i, j int) bool {
	return n[i].distance < n[j].distance
}

// Tradition struct for unmarshalling json and serving motif distributions
type Tradition struct {
	Name      string
	Latitude  float64
	Longitude float64
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
	ntrads := -1
	var err error
	if n, ok := query["num"]; ok && len(n) == 1 {
		ntrads, err = strconv.Atoi(n[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	if ntrads == -1 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
	data := make([]map[string]string, 0)
	for _, val := range q.getNNearestNeighbours(code[0], ntrads) {
		data = append(data, map[string]string{
			"code":     val.code,
			"distance": strconv.FormatFloat(val.distance, 'f', 5, 64),
		})
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

// Generic closure for fetching json data; used only once here,
// but may be useful later
func createJSONServer(data []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "Bad request: %s", r.URL.Path[1:])
}

func main() {
	err := os.Chdir("/root/mythology-queries/bin")
	if err != nil {
		log.Fatal(err)
	}
	// Prepare tradition data for handling traditionQueries
	traditionsRaw, err := ioutil.ReadFile("../data/traditions.json")
	if err != nil {
		log.Fatal(err)
	}
	traditionDict := make(map[string][]int)
	err = json.Unmarshal(traditionsRaw, &traditionDict)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare motif data for handling fetchMotifDistr queries
	motifsRaw, err := ioutil.ReadFile("../data/motif_distributions.json")
	if err != nil {
		log.Fatal(err)
	}
	motifDict := make(map[string][]int)
	err = json.Unmarshal(motifsRaw, &motifDict)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare motif representations for motifQueries
	motifVectorsRaw, err := ioutil.ReadFile("../data/motif_vectors.json")
	if err != nil {
		log.Fatal(err)
	}
	motifVectors := make(map[string][]int)
	err = json.Unmarshal(motifVectorsRaw, &motifVectors)
	if err != nil {
		log.Fatal(err)
	}
	motifWeights := make(map[string]float64)
	for key, vector := range motifVectors {
		sum := 0
		for _, val := range vector {
			sum += val
		}
		motifWeights[key] = 1.0 / float64(sum)
	}

	// Prepare and serve the motif list; the list will be used later
	// for comparing traditions
	motifList := []string{}
	motifListBytes, err := ioutil.ReadFile("../data/motif_list.json")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/fetchMotifList", createJSONServer(motifListBytes))
	err = json.Unmarshal(motifListBytes, &motifList)
	if err != nil {
		log.Fatal(err)
	}

	// Read coords to a slice; use it for distance calculations
	coords := []Tradition{}
	coordsBytes, err := ioutil.ReadFile("../data/coords.json")
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(coordsBytes, &coords)

	// TODO: serve from coords slice
	// Serve the tradition info straight from the file
	http.HandleFunc("/fetchTraditionDict", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, "../data/coords.json")
	})

	// Prepare and serve motif distributions
	traditions := []Tradition{}
	traditionListRaw, err := ioutil.ReadFile("../data/coords.json")
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(traditionListRaw, &traditions)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/fetchMotifDistr", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		query := r.URL.Query()
		code, ok := query["code"]
		// Check that a code is present and that there is only of them.
		if !ok || len(code) != 1 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if _, ok = motifDict[code[0]]; !ok {
			http.Error(w, "Wrong motif code", http.StatusNotFound)
			return
		}
		traditionsForMotif := []Tradition{}
		for idx, val := range motifDict[code[0]] {
			if val == 1 {
				traditionsForMotif = append(traditionsForMotif, traditions[idx])
			}
		}
		distributionData, err := json.Marshal(traditionsForMotif)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(distributionData)
	})

	// Initialise motifHandler with the data and register it with the server
	motifHandler := queryHandler{}
	motifHandler.distance = func(code1, code2 string) (float64, error) {
		v1 := motifDict[code1]
		v2 := motifDict[code2]
		if len(v1) != len(v2) {
			return -1.0, errors.New("Vectors must be of the same length")
		}
		var distance float64
		distance = 0
		// Distance is defined as the maximum distance from a point in one of
		// the vectors to the nearest point in the other one. It is computed
		// separately for both vectors, but since we take the max it is symmetric.
		for idx1, val1 := range v1 {
			if val1 == 0 {
				continue
			}
			minDist := math.Inf(1)
			p1 := s2.LatLngFromDegrees(coords[idx1].Latitude, coords[idx1].Longitude)
			for idx2, val2 := range v2 {
				if val2 == 0 {
					continue
				}
				p2 := s2.LatLngFromDegrees(coords[idx2].Latitude, coords[idx2].Longitude)
				dist := float64(p1.Distance(p2))
				if dist < minDist {
					minDist = dist
				}
			}
			if minDist > distance {
				distance = minDist
			}
		}
		// And now the other way around.
		for idx1, val1 := range v2 {
			if val1 == 0 {
				continue
			}
			minDist := math.Inf(1)
			p1 := s2.LatLngFromDegrees(coords[idx1].Latitude, coords[idx1].Longitude)
			for idx2, val2 := range v1 {
				if val2 == 0 {
					continue
				}
				p2 := s2.LatLngFromDegrees(coords[idx2].Latitude, coords[idx2].Longitude)
				dist := float64(p1.Distance(p2))
				if dist < minDist {
					minDist = dist
				}
			}
			if minDist > distance {
				distance = minDist
			}
		}
		return distance, nil
	}
	motifHandler.items = make(map[string]bool)
	for key, _ := range motifVectors {
		motifHandler.items[key] = true
	}
	http.HandleFunc("/motifQuery", motifHandler.handleQuery)

	// Initialise traditionHandler with the data and register it with the server
	traditionHandler := queryHandler{}
	traditionHandler.distance = func(code1, code2 string) (float64, error) {
		v1 := traditionDict[code1]
		v2 := traditionDict[code2]
		if len(v1) != len(v2) {
			return -1.0, errors.New("Vectors must be of the same length")
		}
		// Distance is defined as negative similarity
		var distance float64
		distance = 0
		for idx, val := range v1 {
			if val == 1 && val == v2[idx] {
				distance -= motifWeights[motifList[idx]]
			}
		}
		return distance, nil
	}
	traditionHandler.items = make(map[string]bool)
	for key, _ := range traditionDict {
		traditionHandler.items[key] = true
	}
	http.HandleFunc("/traditionQuery", traditionHandler.handleQuery)

	// Initialise the tradition comparison handler
	http.HandleFunc("/compareTraditions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		query := r.URL.Query()
		trad1, ok1 := query["trad1"]
		trad2, ok2 := query["trad2"]
		if !ok1 || !ok2 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		trad1vec, ok1 := traditionDict[trad1[0]]
		trad2vec, ok2 := traditionDict[trad2[0]]
		if !ok1 || !ok2 {
			http.Error(w, "Wrong motif code", http.StatusNotFound)
			return
		}
		result := map[string][]string{
			"common": []string{},
			trad1[0]: []string{},
			trad2[0]: []string{},
		}
		for idx, val := range trad1vec {
			if val == 1 && trad2vec[idx] == 1 {
				result["common"] = append(result["common"], motifList[idx])
			} else if val == 1 {
				result[trad1[0]] = append(result[trad1[0]], motifList[idx])
			} else if trad2vec[idx] == 1 {
				result[trad2[0]] = append(result[trad2[0]], motifList[idx])
			}
		}
		resultData, err := json.Marshal(result)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(resultData)
	})

	// Requests for all other paths are bad by definition
	http.HandleFunc("/", errorHandler)

	// Get to work
	http.ListenAndServe(":8080", nil)
}
