package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {
	var wg sync.WaitGroup
	var processes []*os.Process
	var processesMutex sync.Mutex

	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Cleanup goroutine
	go func() {
		<-signalCh
		log.Println("Cleaning up...")
		processesMutex.Lock()
		for _, proc := range processes {
			fmt.Printf("terminating process: %v\n", proc.Pid)
			err := proc.Signal(syscall.SIGTERM)
			if err != nil {
				log.Printf("Error terminating process: %v\n", err)
			}
		}
		processesMutex.Unlock()
		wg.Wait()
		os.Exit(0)
	}()

	for _, arg := range os.Args[1:] {
		fields := strings.Fields(arg)
		cmd := exec.Command(fields[0], fields[1:]...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Println("Error creating StdoutPipe:", err.Error())
			signalCh <- syscall.SIGINT
			select {}
		}
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := cmd.Start()
			if err != nil {
				log.Println("Error starting command:", err.Error())
				signalCh <- syscall.SIGINT
				select {}
			}
			processesMutex.Lock()
			processes = append(processes, cmd.Process)
			processesMutex.Unlock()

			_, err = io.Copy(os.Stdout, stdout)
			if err != nil {
				log.Println("Error copying output to Stdout", err.Error())
			}

			err = cmd.Wait()
			if err != nil {
				if !isInterrupt(err) {
					log.Println("Error waiting for command:", err)
					signalCh <- syscall.SIGINT
					select {}

				}
			}
		}()
	}
	wg.Wait()
}
func isInterrupt(err error) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.Signal() == os.Interrupt
		}
	}
	return false
}
