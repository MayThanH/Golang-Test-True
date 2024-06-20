package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var RedisClient *redis.Client
var Ctx = context.Background()



func ConnectMongoDB() {
    var err error
    clientOptions := options.Client().ApplyURI("mongodb+srv://thanchanokmth:xbi0UpsxbhQSzkLR@go-mongo-redis-cluster.ookyc4d.mongodb.net/?retryWrites=true&w=majority&appName=go-mongo-redis-cluster")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    MongoClient, err = mongo.Connect(ctx, clientOptions)
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }

    err = MongoClient.Ping(ctx, nil)
    if err != nil {
        log.Fatalf("Failed to ping MongoDB: %v", err)
    }
    fmt.Println("Connected to MongoDB!")
}

func ConnectRedis() {
    RedisClient = redis.NewClient(&redis.Options{
        Addr:    "redis-18253.c302.asia-northeast1-1.gce.redns.redis-cloud.com:18253",
        Password: "iBNY2ZauoBRkGxdw1ziR8TxUJMU0MYID",
    })

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _, err := RedisClient.Ping(ctx).Result()
    if err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }
    fmt.Println("Connected to Redis!")
}