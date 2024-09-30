package main

import (
	"ds-cache/api"
	"ds-cache/api/helper"
	"ds-cache/internal"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func initEnv() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error is occurred  on .env file please check")
	}
}

func initDataNodes() []internal.DataNode {
	numberOfNodes, err := strconv.Atoi(os.Getenv("CACHE_MANAGER_NODES_NUMBER"))

	if err != nil {
		fmt.Println("Error converting CACHE_MANAGER_NODES_NUMBER to integer:", err)
		return nil
	}

	nodes := make([]internal.DataNode, 0)

	for i := 1; i <= numberOfNodes; i++ {
		// TODO: Make the node capacity configurable
		nodes = append(nodes, *internal.NewDataNode("node-"+strconv.Itoa(i), 1000))
	}

	return nodes
}

func initCacheManager() {

	fmt.Printf("Initializing cache manager...\n")

	handlers := api.NewHandler(initDataNodes())
	routes := api.NewRouter(handlers)

	server := &http.Server{
		Addr:           os.Getenv("CACHE_MANAGER_PORT"),
		Handler:        routes,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	err := server.ListenAndServe()
	helper.ErrorPanic(err)
}

func main() {
	initEnv()
	initCacheManager()
}
