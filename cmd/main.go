package main

import (
	"github.com/chaithanyaKS/db-backend/server"
)

func main() {
	server := server.New("localhost", 5555)
	server.ListenAndServe()
}
