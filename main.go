package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"flag"
	"github.com/gorilla/websocket"
	"time"
)

/* Structures */
type Message struct {
	Name string
	Body string
	Time int64
}

/* Initialize vars */
var wg sync.WaitGroup
var addr = flag.String("addr", ":1339", "http service address")

/* Upgraders is part of girolla, holds websocket options */
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/* WEBSOCKET HANDLER */
func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "http://" + r.Host {								// Handle bad origin - otherwise there will be no error
		http.Error(w, "Origin not allowed", 403)
		return
	}

	w.Header().Set("Content-Type", "application/javascript") 							// We're using JSON messages to communicate
	c, err := upgrader.Upgrade(w, r, nil)
	checkError(err)

	// Initialize vars
	isOpen := true
	quit := make(chan bool)												// Create channel for quitting
	msg := new(Message)

	/* Go Routine for  */
	go func() {
		for isOpen {
			err = c.ReadJSON(&msg)										// read incoming message from websocket
			if err != nil {											// Graceful disconnect
				isOpen = false
				log.Print("Client disconnected")
				break
			}

			/* New seed received */
			if (msg.Name == "plant_seed") {									// Spin up goRoutine which generates fractals concurrently,
				go plantSeed(msg, c, quit);								// passing our websocket connection, and a quit channel
			}

			/* Cancel Request received */
			if (msg.Name == "cancel") {
				quit<-true 										// Send quit message over the receive channel we passed in to plantSeed()
				log.Print("Cancel request received")
			}

			log.Printf("Websocket message: %#v %#v\n", msg.Name, msg.Body)					// Log new message. We do this at end so we can see any blocks
		}
	}()
}

func plantSeed(msg *Message, conn *websocket.Conn,quit <-chan bool) {
	log.Print("planting seed " + msg.Body);
	seed, _ := strconv.ParseInt(msg.Body, 10, 64) // Convert seed string to int64
	err := conn.WriteJSON(msg)
	checkError(err)

	cpus := runtime.NumCPU() 					// Get number of CPUs / cores
	cancel1 := make(chan bool,1) // Buffered non-blocking channel
	cancel2 := make(chan bool,1) // Buffered non-blocking channel
	cancel3 := make(chan bool,1) // Buffered non-blocking channel
	cancel4 := make(chan bool,1) // Buffered non-blocking channel

	pct1 := make(chan int,10) // Buffered non-blocking channel
	pct2 := make(chan int,10) // Buffered non-blocking channel
	pct3 := make(chan int,10) // Buffered non-blocking channel
	pct4 := make(chan int,10) // Buffered non-blocking channel


	// Generate fractals concurrently
	wg.Add(cpus)                // Add  4 processes to waitgroup
	i:=0
	for i < cpus {
		log.Print(strconv.Itoa(cpus))
		i++
	}
	generateFractal(1, seed, cancel1, pct1) // Start a go routine
	generateFractal(2, seed, cancel2, pct2) // Start a go routine
	generateFractal(3, seed, cancel3, pct3) // Start a go routine
	generateFractal(4, seed, cancel4, pct4) // Start a go routine

	msg = new(Message)
	done1 := false
	done2 := false
	done3 := false
	done4 := false
	for {														// Blocks until all 4 done messages are received
		select {
		case cpct1 := <-pct1:											// Send Pct 1
			msg.Name = "pct1"
			msg.Body = strconv.Itoa(cpct1)
			err = conn.WriteJSON(msg)
			checkError(err)
			if cpct1 == 100 {
				done1 = true										// Pct 1 Done
			}
		case cpct2 := <-pct2:
			msg.Name = "pct2"
			msg.Body = strconv.Itoa(cpct2)
			err = conn.WriteJSON(msg)
			checkError(err)
			if cpct2 == 100 {
				done2 = true
			}
		case cpct3 := <-pct3:
			msg.Name = "pct3"
			msg.Body = strconv.Itoa(cpct3)
			err = conn.WriteJSON(msg)
			checkError(err)
			if cpct3 == 100 {
				done3 = true
			}
		case cpct4 := <-pct4:
			msg.Name = "pct4"
			msg.Body = strconv.Itoa(cpct4)
			err = conn.WriteJSON(msg)
			checkError(err)
			if cpct4 == 100 {
				done4 = true
			}
		case <-quit:												// Quit received from channel, sent it upstream
			cancel1<-true
			cancel2<-true
			cancel3<-true
			cancel4<-true
			//done1 = true
			//done2 = true
			//done3 = true
			//done4 = true
			return
		}

		if done1 == true && done2 == true && done3 == true && done4 == true {
			go func() {
				time.Sleep(time.Second)
				close(pct1)
				close(pct2)
				close(pct3)
				close(pct4)
				log.Print("That was an obnoxious amount of math. Gophers are dead.")
				//panic("How many were running?")
			}()
			break
		}
	}
	wg.Wait()
	// Done
	/* Fractal Done Generating - Send 100% */
	fmt.Printf("Done\n") // Output to console
}

func main() {
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/ws", wsHandler)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}