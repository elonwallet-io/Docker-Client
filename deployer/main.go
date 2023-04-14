package main

import (
	"backend/kubernetes_client/server"
	"fmt"
)

func main() {
	server, err := server.New()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = server.Run()
	if err != nil {
		fmt.Println(err)
		return
	}
}
