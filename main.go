package main

import (
	"io"
	"os/exec"
	"sync"
	"syscall"

	"github.com/dihedron/go-log"
	"github.com/fatih/color"
)

type writef func(format string, args ...interface{}) (int, error)
type writeln func(args ...interface{}) (int, error)

var (
	StdOutF  writef  = color.New(color.FgGreen).Printf
	StdOutLn writeln = color.New(color.FgGreen).Println
	StdErrF  writef  = color.New(color.FgRed).Printf
	StdErrLn writeln = color.New(color.FgRed).Println
)

// The main goroutine will spawn the external process; the external process will
// start producing its outputs to both StdOut and StdErr, which we want to collect
// while we're waiting for the external process to complete. In order not to be
// blocked waiting for one of StdOut and StdErr to become ready, we must do the
// waiting and reading in separate goroutines, one per stream, and move on to
// waiting for the process to terminate; the two goroutines will produce a series
// of events and output them on their respective channel; another goroutine will
// wait for the process to terminate and output yet anothe event to a third channel.
// Thus, the main goroutine will be able to move on and be able to process any
// event (be it something available on StdOut, StdErr or the process having
// terminated) with a single handler.

func read(stream io.Reader, data chan<- []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infof("opening stream")
	buffer := make([]byte, 64)
	for {
		n, err := stream.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Debugf("read to EOF (%d bytes)", n)
				data <- buffer[:n]
			} else {
				log.Errorf("error reading from pipe: %v", err)
			}
			break
		}
		log.Debugf("read data (%d bytes)", n)
		data <- buffer[:n]
	}
	log.Infof("closing stream")
	close(data)
}

func wait(command *exec.Cmd, done chan<- bool, wg *sync.WaitGroup) {
	defer close(done)
	defer wg.Done()
	if err := command.Wait(); err != nil {
		log.Errorf("error waiting for process to complete: %v", err)
		done <- false
	}
	done <- true
}

func main() {
	var wg sync.WaitGroup
	syscall.Clearenv()
	syscall.Setenv("MYKEY", "MYVALUE")
	command := exec.Command("/bin/bash", "-c", "ls -a -l -h")

	err := command.Start()
	if err != nil {
		log.Fatalf("error starting command: %v", err)
	}

	done := make(chan bool)
	wg.Add(1)
	go wait(command, done, &wg)
	select {
	case complete, ok := <-done:
		log.Infof("complete: %t, ok: %t", complete, ok)
	default:
		log.Debugf("in default")
	}

	//wg.Wait()
}

func main2() {

	var wg sync.WaitGroup

	syscall.Clearenv()
	syscall.Setenv("MYKEY", "MYVALUE")
	command := exec.Command("/bin/bash", "-c", "./test.sh")

	stdin, _ := command.StdinPipe()
	// stdout, _ := command.StdoutPipe()
	// stderr, _ := command.StderrPipe()

	stdout := make(chan []byte)
	stderr := make(chan []byte)
	done := make(chan bool)

	s, err := command.StdoutPipe()
	if err != nil {
		log.Fatalf("error opening STDOUT pipe: %v", err)
	}
	wg.Add(1)
	go read(s, stdout, &wg)

	s, err = command.StderrPipe()
	if err != nil {
		log.Fatalf("error opening STDERR pipe: %v", err)
	}
	wg.Add(1)
	go read(s, stderr, &wg)

	err = command.Start()
	if err != nil {
		log.Fatalf("error starting command: %v", err)
	}

	wg.Add(1)
	go wait(command, done, &wg)

	stdin.Write([]byte("a\nb\nc\n"))
	stdin.Close()

	var stdOutClosed, stdErrClosed, programTerminated bool
outer:
	for {
		select {
		case data, ok := <-stdout:
			if ok {
				log.Infof("STDOUT received: %q", string(data))
			} else {
				if !stdOutClosed {
					stdOutClosed = true
					log.Warnln("no more stdout (channel closed)")
				}
			}
		case data, ok := <-stderr:
			if ok {
				log.Infof("STDERR received: %q", string(data))
			} else {
				if !stdErrClosed {
					stdErrClosed = true
					log.Warnln("no more stderr (channel closed)")
				}
			}
		case complete, ok := <-done:
			if ok {
				log.Infof("job complete with %t", complete)
				break
			} else {
				if !programTerminated {
					programTerminated = true
					log.Warnln("no completion (channel closed)")
				}
			}
		default:
			if stdOutClosed && stdErrClosed && programTerminated {
				log.Infof("closing down")
				break outer
			}
			log.Debugln("still looping...")
		}
	}

	wg.Wait()
	//buffer, _ := ioutil.ReadAll(stdout)
	//command.Wait()

	// fmt.Println(string(buffer))

	// lsCmd := exec.Cmmand("bash", "-c", "ls -a -l -h")
	// lsOut, err := lsCmd.Output()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("> ls -a -l -h")
	// fmt.Println(string(lsOut))
}
