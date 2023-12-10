package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	var (
		wg             sync.WaitGroup
		processes      []*exec.Cmd
		processesMutex sync.Mutex
		once           sync.Once
	)
	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Cleanup goroutine
	go func() {
		<-signalCh
		log.Println("Cleaning up...")
		time.Sleep(time.Millisecond * 100)
		processesMutex.Lock()
		for _, proc := range processes {
			log.Printf("%v Exiting Command with process PID %d\n", proc, proc.Process.Pid)
			if err := proc.Process.Signal(os.Interrupt); err != nil {
				log.Printf("Error sending Interrupt signal to process: %v", err)
			}

			if err := proc.Process.Signal(syscall.SIGTERM); err != nil {
				log.Printf("Error sending SIGTERM signal to process: %v", err)
			}

			if err := proc.Process.Signal(syscall.SIGINT); err != nil {
				log.Printf("Error sending SIGINT signal to process: %v", err)
			}
		}
		processesMutex.Unlock()
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
				processes = append(processes, cmd)
				processesMutex.Unlock()

				if _, err := io.Copy(os.Stdout, stdout); err != nil && !errors.Is(err, io.EOF) {
					return fmt.Errorf("error copying output to Stdout: %w", err)
				}

				if err := cmd.Wait(); err != nil {
					return fmt.Errorf("error waiting for command: %w", err)
				}
				return nil
			}()
			once.Do(func() {
				if err != nil {
					log.Println("encountered an error -> exiting:", err)
				} else {
					log.Println("program terminated -> exiting")
				}
				signalCh <- syscall.SIGTERM
				runtime.Gosched()

			})

		}(arg)
	}
	wg.Wait()
}
