package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"statecheck/registry"
)

func main() {
	args := os.Args
	if err := checkArgsLen(&args); err != nil {
		log.Fatal(err)
	}
	url := args[1]

	client := &http.Client{Timeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	go func() { <-ctx.Done(); stop() }()
	registry := registry.NewRegistry(ctx, client)

	// some kind of testing
	var err error
	err = registry.Add("ps1", url)
	if err != nil {
		fmt.Println(err)
	}
	err = registry.Poll("ps1")
	if err != nil {
		fmt.Println(err)
	}
	// time.Sleep(7 * time.Second)
	fmt.Println("")
	err = registry.StopPolling("ps1")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("")
	err = registry.Poll("ps1")
	if err != nil {
		fmt.Println(err)
	}
	// time.Sleep(7 * time.Second)
	err = registry.Delete("ps1")
	if err != nil {
		fmt.Println(err)
	}

	<-ctx.Done()
}

func checkArgsLen(args *[]string) error {
	if len(*args) == 1 {
		return fmt.Errorf("Needs 1 url")
	}
	if len(*args) != 2 {
		return fmt.Errorf("Needs only 1 url")
	}
	return nil
}
