package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"tester/internal/db"
	"tester/pkg/memory"

	"github.com/joho/godotenv"
)


var numWorkers int
var workers []chan int 


func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	totalRecordsStr := os.Getenv("TOTAL_RECORDS")
	if totalRecordsStr == "" {
		log.Fatal("TOTAL_RECORDS not set in environment")
	}
	fmt.Println("recorders to insert :",totalRecordsStr )
	totalRecords, err := strconv.Atoi(totalRecordsStr)
	if err != nil {
		log.Fatalf("Invalid TOTAL_RECORDS value: %s", err)
	}

	ctx := context.Background()
	milvusClient, err := db.ConectToDb(ctx)
	if err != nil {
		log.Fatalf("error connecting to db %s", err)
	}
	defer milvusClient.Close()

	start := time.Now()
	var wg sync.WaitGroup

	numWorkersStr := os.Getenv("WORKERS")
	if numWorkersStr == "" {
		log.Fatal("WORKERS not set in environment")
	}
	numWorkers, err = strconv.Atoi(numWorkersStr)
	if err != nil {
		log.Fatal("couldnt convert numworkers to int")
	}

	fmt.Println("goroutine numbers : ", numWorkers)

	workers = make([]chan int, numWorkers)

	recordsPerWorker := totalRecords / numWorkers
	workChan := make(chan int, numWorkers)
	// inedx , _ := db.GetCollectionRecordCount(context.Background(), milvusClient, "random_vectors")
	// panic(inedx)
	// _ = db.DropAndRecreateCollection(ctx,milvusClient, "random_vectors" )
	// panic("stop")

	err = db.CreateCollectionIfNotExists(context.Background(), milvusClient, "random_vectors", numWorkers)
	if err != nil {
		log.Fatal("error creating collection : ", err)
	}

	go memory.MonitorMemoryUsage(workers)

	
	for i := 0; i < numWorkers; i++ {
		workers[i] = make(chan int, 1)
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range workChan {
				err := db.InsertRecords(ctx, milvusClient, "random_vectors", job, recordsPerWorker, workers[workerID])
				if err != nil {
					panic(err)
				}
			}
		}(i)
	}

	for i := 0; i < numWorkers; i++ {
		workChan <- i
	}

	close(workChan)
	wg.Wait()

	end := time.Since(start)
	fmt.Printf("All records inserted successfully! Application runtime: %v\n", end)
}


