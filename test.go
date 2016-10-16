// package main

// import (
// 	"fmt"
// 	"io"
// 	"io/ioutil"
// 	// "os"
// )

// func main() {

// 	a := make(chan int)

// 	r, w := io.Pipe()

// 	go func() {
// 		// 	a <- 1
// 		for {

// 			w.Write([]byte("hello"))
// 		}

// 		fmt.Println("Print")
// 	}()

// 	go func() {
// 		for {
// 			test, _ := r.Read(data)
// 			fmt.Println(test)

// 		}
// 	}()

// 	for {
// 		_ = <-a
// 		return
// 	}
// }
