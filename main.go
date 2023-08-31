package awesomeTest

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

type SDNList struct {
	XMLName xml.Name `xml:"sdnList"`
	Records []Record `xml:"sdnEntry"`
}

type Record struct {
	UID        string `xml:"uid"`
	SDNType    string `xml:"sdnType"`
	FirstName  string `xml:"firstName"`
	LastName   string `xml:"lastName"`
	// ... else
}

func main() {
	// Registrasia obrobotchikov
	http.HandleFunc("/update", update)
	http.HandleFunc("/state", state)

	// zapusk servera
	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}

var dataState string = "empty" // Изначально состояние "empty"

// obnovlenia dannyh
func update(w http.ResponseWriter, r *http.Request) {
	dataState = "updating"
	
	resp, err := http.Get("https://www.treasury.gov/ofac/downloads/sdn.xml")
	
	if err != nil {
		unsuccessfull(w)
		return
	}
	defer resp.Body.Close()

	var sdnList SDNList
	
	decoder := xml.NewDecoder(resp.Body)
	
	err = decoder.Decode(&sdnList)
	
	if err != nil {
		unsuccessfull(w)
		return
	}

	db, _ := sql.Open("postgres", "user=dbuser dbname=dbname sslmode=disable")

	defer db.Close()

	// Create a prepared statement for batch inserts
	stmt, err := db.Prepare("INSERT INTO individuals (uid, first_name, last_name) VALUES ($1, $2, $3)")
	
	if err != nil {
		unsuccessfull(w)
		return
	}
	defer stmt.Close()

	// Batch insert
	for _, record := range sdnList.Records {
		if record.SDNType == "Individual" {
			_, err := stmt.Exec(record.UID, record.FirstName, record.LastName)
			
			if err != nil {
				http.Error(w, "Error inserting record", http.StatusInternalServerError)
				
				return
			}
		}
	}
	dataState = "ok"
	
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]string{
		"result": strconv.FormatBool(true),
		"info": "",
	})
	
}

func state(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]string{
		"result": strconv.FormatBool(dataState=="ok"),
		"info": dataState,
	})

}
func getNames(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	name := query.Get("name")

	namesType := query.Get("type")

	var records []Record

	db, _ := sql.Open("postgres", "user=dbuser dbname=dbname sslmode=disable")

	defer db.Close()

	sqlQuery:="SELECT uid, first_name, last_name FROM individuals WHERE first_name = $1 OR last_name = $1"

	if namesType != "strong"{
		sqlQuery = "SELECT uid, first_name, last_name FROM individuals WHERE first_name LIKE '%' || $1 || '%' OR last_name LIKE '%' || $1 || '%'"
	}

	rows, err := db.Query(sqlQuery, name)

	if err != nil {
	  http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
	  return
	}

	defer rows.Close()

	for rows.Next() {
	  var record Record
	  err := rows.Scan(&record.UID, &record.FirstName, &record.LastName)
	  if err != nil {
	    http.Error(w, "Failed to scan data", http.StatusInternalServerError)
	    return
	  }
	  records = append(records, record)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(records)
}

func unsuccessfull(w http.ResponseWriter){
	w.WriteHeader(http.StatusServiceUnavailable)

	json.NewEncoder(w).Encode(map[string]string{
		"result": strconv.FormatBool(false),
		"info": "service unavailable",
	})
}