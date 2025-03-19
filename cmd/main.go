package main

import (
	"github.com/chaithanyaKS/postgres-backend/server"
)

func main() {
	server := server.New("localhost", 5555)
	server.ListenAndServe()
}
