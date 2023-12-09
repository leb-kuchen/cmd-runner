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
	"sync/atomic"
	"syscall"
)

func main() {
	var (
		wg             sync.WaitGroup
		processes      []*os.Process
		processesMutex sync.Mutex
		errFound       atomic.Bool
	)
	signalCh := make(chan os.Signal, 3)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Cleanup goroutine
	go func() {
		<-signalCh
		log.Println("Cleaning up...")
		processesMutex.Lock()
		for _, proc := range processes {
			log.Printf("terminating process: %v\n", proc.Pid)
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
		arg := arg
		wg.Add(1)
		go func(arg string) {
			defer wg.Done()
			err := func() error {
				fields := strings.Fields(arg)
				cmd := exec.Command(fields[0], fields[1:]...)
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					return fmt.Errorf("error creating StdoutPipe: %w", err)
				}
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin

				if err := cmd.Start(); err != nil {
					return fmt.Errorf("error starting command: %w", err)
				}
				processesMutex.Lock()
				processes = append(processes, cmd.Process)
				processesMutex.Unlock()

				if _, err := io.Copy(os.Stdout, stdout); err != nil && !errors.Is(err, io.EOF) {
					return fmt.Errorf("error copying output to Stdout: %w", err)
				}

				if err := cmd.Wait(); err != nil && !isInterrupt(err) {
					return fmt.Errorf("error waiting for command: %w", err)
				}
				return nil
			}()
			if err != nil {
				log.Println("encountered an error -> exiting:", err)
				signalCh <- syscall.SIGTERM
				errFound.Store(true)
			}
		}(arg)
		if errFound.Load() {
			continue
		}
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
