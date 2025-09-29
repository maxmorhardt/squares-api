package main

import "squares-api/internal/routes"

func main() {
	r := routes.SetupRouter()
	r.Run(":8080")
}