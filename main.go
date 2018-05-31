package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func read(stream io.Reader, data chan<- []byte, done <-chan bool) {

	buffer := make([]byte, 4)
	for {
		select {
		case quit := <-done:
			fmt.Println("received quit message", quit)
			return
		default:
			fmt.Println("no quit message received")
			for {
				n, err := stream.Read(buffer)
				if err != nil {
					if err == io.EOF {
						fmt.Println(string(buffer[:n])) //should handle any remainding bytes.
						break
					}
					fmt.Println(err)
					os.Exit(1)
				}
				data <- buffer[:n]
			}
		}
	}
}

func main() {

	syscall.Clearenv()
	syscall.Setenv("MYKEY", "MYVALUE")
	command := exec.Command("/bin/bash", "-c", "./test.sh")

	stdin, _ := command.StdinPipe()
	// stdout, _ := command.StdoutPipe()
	// stderr, _ := command.StderrPipe()

	stdout := make(chan []byte)
	stderr := make(chan []byte)
	done := make(chan bool)

	err := command.Start()
	if err != nil {
		log.Fatal(err)
		fmt.Printf("Error: %v\n", err)
	}
	s, _ := command.StdoutPipe()
	go read(s, stdout, done)
	stdin.Write([]byte("a\nb\nc\n"))
	stdin.Close()
	buffer, _ := ioutil.ReadAll(stdout)
	command.Wait()

	fmt.Println(string(buffer))

	// lsCmd := exec.Cmmand("bash", "-c", "ls -a -l -h")
	// lsOut, err := lsCmd.Output()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("> ls -a -l -h")
	// fmt.Println(string(lsOut))
}
