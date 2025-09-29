package main

import "github.com/maxmorhardt/squares-api/internal/routes"

func main() {
	routes.NewRouter().Run(":8080")
}