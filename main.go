package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

var db *sql.DB

type response struct {
	AccountID   int    `json:"accountId"`
	Balance     int    `json:"balance"`
	CreatedAt   string `json:"created_at"`
	ClusterName string `json:"clusterName"`
}

func addDeposit(w http.ResponseWriter, r *http.Request) {

	// Create the "accounts" table.
	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS accounts (id SERIAL PRIMARY KEY, balance INT, cluster_name CHARACTER(10), created_at TIMESTAMP DEFAULT NOW())"); err != nil {
		log.Fatal(err)
	}

	sql := fmt.Sprintf("INSERT INTO accounts (cluster_name, balance) VALUES ('%s', 250)", os.Getenv("CLUSTER_NAME"))

	log.Info(sql)

	if _, err := db.Exec(sql); err != nil {
		log.Fatal(err)
	}

}

func getLatestDeposit(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query("SELECT id, balance, created_at, cluster_name FROM accounts WHERE id = (SELECT MAX(id) FROM accounts)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Initial balances:")

	var id, balance int
	var createdat, clustername string
	for rows.Next() {

		if err := rows.Scan(&id, &balance, &createdat, &clustername); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d %d %s %s\n", id, balance, createdat, clustername)
	}

	response := new(response)
	response.AccountID = id
	response.Balance = balance
	response.CreatedAt = createdat
	response.ClusterName = clustername

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

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/bank/deposit", addDeposit)
	router.HandleFunc("/bank/deposit/show", getLatestDeposit)
	log.Fatal(http.ListenAndServe(":8080", router))

}
