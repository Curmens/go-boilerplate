package main

import (
	"example.com/config"
	"example.com/routes"
)

func main() {
	//database.ConnectDb()

	// load configs
	appConfig, _ := config.Load()
	routes.SetupRouter(appConfig.Logger)
}
