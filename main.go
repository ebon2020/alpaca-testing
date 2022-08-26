package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v2/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/v2/marketdata/stream"
	"github.com/joho/godotenv"
)

func askForConfirmation(question string) bool {
	var option string
	fmt.Printf("Please confirm - '%v?': ", question)
	_, err := fmt.Scan(&option)
	if err != nil {
		log.Fatal("Error parsing")
	}

	option = strings.ToLower(option)

	switch option {
	case "y", "yes":
		fmt.Println("Continuing...")
		return true
	case "n", "no":
		fmt.Println("Skipping...")
		return false
	default:
		fmt.Println("Not a valid response... (Yes/No, Y/N)")
		return askForConfirmation(question)
	}
}

func init() {
	err := godotenv.Load("apiKeys.env")
	if err != nil {
		log.Fatal("Error loading API keys")
	} else {
		customApiKey := os.Getenv("apiKey")
		customApiSecret := os.Getenv("apiSecret")
		log.Print("API Keys loaded!")
		fmt.Printf("apiKey: %v\napiSecret: %v\n", customApiKey, customApiSecret)
	}

	eClient := alpaca.NewClient(alpaca.ClientOpts{
		ApiKey:    os.Getenv("apiKey"),
		ApiSecret: os.Getenv("apiSecret"),
		BaseURL:   "https://paper-api.alpaca.markets",
	})

	account, err := eClient.GetAccount()
	if err != nil {
		panic(err)
	} else {
		fmt.Printf("Your account currently has $%v of buying power and $%v in liquid assets\n", account.BuyingPower, account.Cash)
	}

	if askForConfirmation("clear account orders") {
		eClient.CancelAllOrders()
	}

	position, err := eClient.ListPositions()
	if err != nil {
		panic(err)
	} else {
		for _, p := range position {
			fmt.Printf("%v %v %v\n", p.Symbol, p.Qty, p.Side)
		}
	}
	fmt.Println(position)
}

func main() {
	var barCount int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	barHandler := func(bar stream.Bar) {
		atomic.AddInt32(&barCount, 1)
	}

	dataStream := stream.NewStocksClient(
		"iex",
		stream.WithCredentials(os.Getenv("apiKey"), os.Getenv("apiSecret")),
		stream.WithBars(barHandler, "AAPL", "SPY"),
	)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Printf("Bars received: %v\n", barCount)
		}
	}()

	if err := dataStream.Connect(ctx); err != nil {
		log.Fatalf("%s", err)
	}
	fmt.Println("Connection Established!")

	go func() {
		err := <-dataStream.Terminated()
		if err != nil {
			log.Fatalf("Program exited with error '%s'", err)
		}
		fmt.Println("Exiting routine...")
		os.Exit(0)
	}()

	// Wait for the stream to be terminated
	<-ctx.Done()

	// Close the stream by pressing Ctrl+C

}
