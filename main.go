package main

import (
	"example.com/database"
	"example.com/routes"
)

func main() {
	database.ConnectDb()
	routes.SetupRouter().Run()
}
