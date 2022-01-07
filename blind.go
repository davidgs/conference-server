package main

import (
	"fmt"
	"log"
)



// let's get this party started!
func main() {
	a := App{}
	err := a.Initialize()
	if err != nil {
		fmt.Println("Error:", err)
		log.Fatal(err)
	}
	a.InitializeRoutes()
	fmt.Println("Server Started")
	a.Run(":3030")
}


