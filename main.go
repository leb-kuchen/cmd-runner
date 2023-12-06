package main

import (
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

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalCh
		fmt.Println("\nReceived interrupt signal. Cleaning up...")
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
			log.Println("Error creating StdoutPipe:", err)
			return
		}
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := cmd.Start()
			if err != nil {
				log.Println("Error starting command:", err)
				return
			}
			processesMutex.Lock()
			processes = append(processes, cmd.Process)
			processesMutex.Unlock()

			io.Copy(os.Stdout, stdout)

			err = cmd.Wait()
			if err != nil {
				log.Println("Error waiting for command:", err)
			}
		}()
	}
	wg.Wait()

}
