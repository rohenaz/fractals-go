package main

import (

	"log"
	"os"
)

// Change this to turn on stack info on error (panic)
const debug bool = true

func checkError(err error) {

	if err != nil {
		if !debug {
			log.Printf("Fatal error: %s\n", err.Error())
			os.Exit(1)
		}else {
			panic("Panic! %s\n"+ err.Error())
		}

	}
}
