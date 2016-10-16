package main

import (
	"io"
	"os"
)

func main() {

	// a := make(chan int)

	r, w := io.Pipe()

	go func() {
		// 	a <- 1
		io.Copy(os.Stdout, r)

	}()

	// for {
	// 	_ = <-a
	// 	return
	// }

	w.Write([]byte("hello"))
}
