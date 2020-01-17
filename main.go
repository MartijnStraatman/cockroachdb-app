package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
)

var db *sql.DB

type response struct {
	TransactionID     int    `json:"transactionId"`
	Amount            int    `json:"amount"`
	CreatedAtTime     string `json:"created_at"`
	SavedByCluster    string `json:"savedByCluster"`
	ResponseByCluster string `json:"responseByCluster"`
}

type deposit struct {
	Amount         int    `json:"amount"`
	SavedByCluster string `json:"savedByCluster"`
}

func addDeposit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Infof("received request")

	reqBody, err := ioutil.ReadAll(r.Body)
	var newDeposit deposit
	newDeposit.SavedByCluster = os.Getenv("CLUSTER_NAME")

	log.Info(reqBody)

	if err != nil {
		log.Error("no request data")
	}

	json.Unmarshal(reqBody, &newDeposit)

	// Create the "accounts" table.
	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS deposit (id SERIAL PRIMARY KEY, amount INT, savedbycluster CHARACTER(10), created_at TIMESTAMP DEFAULT NOW())"); err != nil {
		log.Fatal(err)
	}

	sql := fmt.Sprintf("INSERT INTO deposit (savedbycluster, amount) VALUES ('%s', '%d')", os.Getenv("CLUSTER_NAME"), newDeposit.Amount)

	log.Info(sql)

	if _, err := db.Exec(sql); err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newDeposit)

	log.Infof("saved %d ", newDeposit.Amount)
}

func getLatestDeposit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Infof("received request")

	rows, err := db.Query("SELECT id, amount, created_at, savedbycluster FROM deposit WHERE id = (SELECT MAX(id) FROM deposit)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Initial balances:")

	var id, amount int
	var createdat, savedbycluster string
	for rows.Next() {

		if err := rows.Scan(&id, &amount, &createdat, &savedbycluster); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d %d %s %s\n", id, amount, createdat, savedbycluster)
	}

	response := new(response)
	response.TransactionID = id
	response.Amount = amount
	response.CreatedAtTime = createdat
	response.SavedByCluster = savedbycluster
	response.ResponseByCluster = os.Getenv("CLUSTER_NAME")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

}

func main() {
	log.Info("starting...")
	log.Info("connecting to database")

	var err error

	db, err = sql.Open("postgres", "postgresql://maxroach@cockroachdb-public:26257/bank?ssl=true&sslmode=require&sslrootcert=/certs/ca.crt&sslkey=/certs/client.maxroach.key&sslcert=/certs/client.maxroach.crt")

	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}
	defer db.Close()

	log.Info("connected to database")

	router := mux.NewRouter().StrictSlash(true)

	api := router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/deposit", addDeposit).Methods(http.MethodPost)
	api.HandleFunc("/deposit", getLatestDeposit).Methods(http.MethodGet)
	log.Fatal(http.ListenAndServe(":8080", router))

}
