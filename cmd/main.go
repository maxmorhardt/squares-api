package main

import "squares-api/internal/routes"

func main() {
	r := routes.SetupRouter()
	r.Run("127.0.0.1:8080")
}