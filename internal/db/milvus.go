package db

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/joho/godotenv"
	milvus "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


var (
	batchSize = 90_000
)

func ConectToDb(ctx context.Context) (milvus.Client, error){


	
	const maxMessageSize = 67 * 1024 * 1024 

	grpcOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize), 
			grpc.MaxCallSendMsgSize(maxMessageSize), 
		),
	}

	

	err := godotenv.Load()
	if err != nil {
		return nil , fmt.Errorf("error loading .env file: %v", err)
	}

	Address := os.Getenv("ADDRESS")
	if Address == "" {
		return nil , fmt.Errorf("TOTAL_RECORDS not set in environment")
	}
	fmt.Println("Connecting to Milvus at:", Address)


	
	config := milvus.Config{
		Address:     Address,
		DialOptions: grpcOptions,
	}

	
	client, err := milvus.NewClient(ctx, config)
	if err != nil {
		return nil , fmt.Errorf("failed to create Milvus client: %v", err)
	}
	

	fmt.Println("Connected to Milvus")
	return client, err
}
func DropAndRecreateCollection(ctx context.Context, milvusClient milvus.Client, collectionName string) error {
	
	hasCollection, _ := milvusClient.HasCollection(ctx, collectionName)
	// if err != nil {
	// 	return fmt.Errorf("failed to check collection existence: %v", err)
	// }
	if hasCollection{
		err := milvusClient.DropCollection(ctx, collectionName)
		if err != nil {
			return fmt.Errorf("failed to drop collection: %v", err)
		}
	
		// schema := &entity.Schema{
		// 	CollectionName: collectionName,
		// 	Description:    "random vectors collection",
		// 	AutoID:         false,
		// 	Fields: []*entity.Field{
		// 		{
		// 			Name:     "id",
		// 			DataType: entity.FieldTypeInt64,
		// 			PrimaryKey: true,
		// 			AutoID: true,
		// 		},
		// 		{
		// 			Name:     "vector",
		// 			DataType: entity.FieldTypeFloatVector,
		// 			TypeParams: map[string]string{
		// 				"dim": "150", 
		// 			},
		// 		},
		// 	},
		// }
	
		// err = milvusClient.CreateCollection(ctx, schema, 2) // 2 shards
		// if err != nil {
		// 	return fmt.Errorf("failed to recreate collection: %v", err)
		// }	
	}else{
		fmt.Println("doesnt exist !")
	}
	
	return nil
}
func WaitForMilvus(ctx context.Context, milvusClient milvus.Client) error {
	maxRetries := 10
	retryDelay := time.Second * 5

	for i := 0; i < maxRetries; i++ {
		
		_, err := milvusClient.ListCollections(ctx)
		if err == nil {
			fmt.Println("Milvus Proxy is ready")
			return nil
		}

		fmt.Printf("Milvus Proxy not ready yet, retrying in %v seconds... (attempt %d/%d)\n", retryDelay.Seconds(), i+1, maxRetries)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("milvus Proxy is not ready after %d retries", maxRetries)
}
func InsertRecords(ctx context.Context, milvusClient milvus.Client, collectionName string, workerID, recordsPerWorker int, ws <-chan int) error {

	partitionName := fmt.Sprintf("partition%d", workerID)
	vectors := make([][]float32, 0, batchSize)
	ides := make([]int64, 0, batchSize)

	for i := 0; i < recordsPerWorker; i++ {
		
		id := int64(workerID * recordsPerWorker+ i)
		ides = append(ides, id)
		vec := make([]float32, 150)
		for j := 0; j < 150; j++ {
			vec[j] = rand.Float32()
		}
		vectors = append(vectors, vec)

		
		if len(vectors) >= batchSize {
			log.Println("Worker", workerID, " inserting")
			vectorColumn := entity.NewColumnFloatVector("vector", 150, vectors)
			idColumn := entity.NewColumnInt64("id" ,ides)
		
			startInsert := time.Now()
				_, err := milvusClient.Insert(ctx, collectionName, partitionName, idColumn ,vectorColumn)
			if err != nil {
				fmt.Printf("error inserting records %v in goroutine %d\n", err, workerID)
			}
			log.Printf("Insertion took: %v seconds in gorutine %d", time.Since(startInsert).Seconds(), workerID)

			vectors = vectors[:0]
			ides = ides[:0]
			select {
			case state := <-ws:
				if state == 0 {
					fmt.Println("Worker", workerID, "paused due to memory constraints.")
					for state = <-ws; state == 0; state = <-ws {
						fmt.Println("Worker", workerID, "waiting to resume...")
					}
					fmt.Println("Worker", workerID , " resumed")
				}
			default:
				// No pause signal, continue
			}
		}
	}

	
	if len(vectors) > 0 {
		log.Println("Worker", workerID, "inserting")
		vectorColumn := entity.NewColumnFloatVector("vector", 150, vectors)
		idColumn := entity.NewColumnInt64("id" ,ides)
		startInsert := time.Now()
		_, err := milvusClient.Insert(ctx, collectionName, partitionName,idColumn , vectorColumn)
		if err != nil {
			return fmt.Errorf("failed to insert records: %v", err)
		}
		log.Printf("Insertion took: %v seconds", time.Since(startInsert).Seconds())
	}
	log.Println("Worker", workerID, "done inserting.")
	return nil
}
func GetCollectionRecordCount(ctx context.Context, milvusClient milvus.Client, collectionName string) (int64, error) {

	hasCollection, err := milvusClient.HasCollection(ctx, collectionName)
	if err != nil {
		return 0, fmt.Errorf("failed to check collection existence: %v", err)
	}
	if !hasCollection {
		return 0, fmt.Errorf("collection %s does not exist", collectionName)
	}

	
	stats, err := milvusClient.GetCollectionStatistics(ctx, collectionName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection statistics: %v", err)
	}

	
	recordCountStr, ok := stats["row_count"]
	if !ok {
		return 0, fmt.Errorf("row_count not found in collection statistics")
	}

	var recordCount int64
	if _, err := fmt.Sscanf(recordCountStr, "%d", &recordCount); err != nil {
		return 0, fmt.Errorf("failed to parse row_count: %v", err)
	}

	return recordCount, nil
}
func CreateCollectionIfNotExists(ctx context.Context, milvusClient milvus.Client, collectionName string, numPartitions int) error {
	
	hasCollection, err := milvusClient.HasCollection(ctx, collectionName)
	if err != nil {
	 return fmt.Errorf("failed to check collection existence: %v", err)
	}
	if !hasCollection {
		log.Println("Creating collection:", collectionName)
	 schema := &entity.Schema{
	  CollectionName: collectionName,
	  Description:    "A collection of random vectors",
	  Fields: []*entity.Field{
	   {Name: "id", DataType: entity.FieldTypeInt64, PrimaryKey: true, AutoID: false},
	   {Name: "vector", DataType: entity.FieldTypeFloatVector, TypeParams: map[string]string{"dim": "150"}},
	  },
	 }
	 err := milvusClient.CreateCollection(ctx, schema, 2)
	 if err != nil {
	  return fmt.Errorf("failed to create collection: %v", err)
	 }
	}
	
	var partitionNames []string
	for i:=0 ; i<numPartitions ; i++ {
		partitionNames = append(partitionNames, fmt.Sprintf("partition%d", i))
	}
	
	for _, partitionName := range partitionNames {
	 hasPartition, err := milvusClient.HasPartition(ctx, collectionName, partitionName)
	 if err != nil {
	  return fmt.Errorf("failed to check partition existence: %v", err)
	 }
	 if !hasPartition {
	  err := milvusClient.CreatePartition(ctx, collectionName, partitionName)
	  if err != nil {
	   return fmt.Errorf("failed to create partition %s: %v", partitionName, err)
	  }
	 }
	}
	
	return nil
}











