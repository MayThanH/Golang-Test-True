package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"golang-api/config"
	"golang-api/models"
)

func CreateItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var item models.Item
	_ = json.NewDecoder(r.Body).Decode(&item)
	item.ID = uuid.New().String()
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	collection := config.MongoClient.Database("testdb").Collection("items")
	_, err := collection.InsertOne(config.Ctx, item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(item)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	itemID := params["id"]

	// Try to get item from Redis cache
	val, err := config.RedisClient.Get(config.Ctx, itemID).Result()
	if err == redis.Nil {
		// If item is not in cache, get from MongoDB
		collection := config.MongoClient.Database("testdb").Collection("items")
		var item models.Item
		err := collection.FindOne(config.Ctx, bson.M{"id": itemID}).Decode(&item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set item to Redis cache
		jsonItem, _ := json.Marshal(item)
		config.RedisClient.Set(config.Ctx, itemID, jsonItem, 5*time.Minute)

		json.NewEncoder(w).Encode(item)
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		// If item is in cache, return it
		var item models.Item
		json.Unmarshal([]byte(val), &item)
		json.NewEncoder(w).Encode(item)
	}
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	itemID := params["id"]
	var item models.Item
	_ = json.NewDecoder(r.Body).Decode(&item)
	item.UpdatedAt = time.Now()
	collection := config.MongoClient.Database("testdb").Collection("items")
	filter := bson.M{"_id": itemID}
	update := bson.M{"$set": bson.M{
		"name":        item.Name,
		"description": item.Description,
		"price":       item.Price,
		"updated_at":  item.UpdatedAt,
	}}
	_, err := collection.UpdateOne(config.Ctx, filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate cache
	config.RedisClient.Del(config.Ctx, itemID)

	json.NewEncoder(w).Encode(item)
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	itemID := params["id"]
	collection := config.MongoClient.Database("testdb").Collection("items")
	filter := bson.M{"_id": itemID}
	_, err := collection.DeleteOne(config.Ctx, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate cache
	config.RedisClient.Del(config.Ctx, itemID)

	json.NewEncoder(w).Encode("Deleted")
}

func GetAllItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Default values for pagination
	limit := 10
	page := 1

	// Get query parameters for limit, page, sorting, and filtering
	query := r.URL.Query()
	if l, ok := query["limit"]; ok {
		limit, _ = strconv.Atoi(l[0])
	}
	if p, ok := query["page"]; ok {
		page, _ = strconv.Atoi(p[0])
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build filter
	filter := bson.M{}
	if name, ok := query["name"]; ok {
		filter["name"] = bson.M{"$regex": name[0], "$options": "i"}
	}
	if description, ok := query["description"]; ok {
		filter["description"] = bson.M{"$regex": description[0], "$options": "i"}
	}
	if minPrice, ok := query["min_price"]; ok {
		if price, err := strconv.ParseFloat(minPrice[0], 64); err == nil {
			filter["price"] = bson.M{"$gte": price}
		}
	}
	if maxPrice, ok := query["max_price"]; ok {
		if price, err := strconv.ParseFloat(maxPrice[0], 64); err == nil {
			filter["price"] = bson.M{"$lte": price}
		}
	}

	// Build sort options
	sort := bson.D{}
	if sortField, ok := query["sort"]; ok {
		sortOrder := 1 // Default to ascending order
		if order, ok := query["order"]; ok && order[0] == "desc" {
			sortOrder = -1
		}
		sort = append(sort, bson.E{Key: sortField[0], Value: sortOrder})
	}

	var items []models.Item
	collection := config.MongoClient.Database("testdb").Collection("items")
	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(offset))
	findOptions.SetSort(sort)

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := cursor.Close(context.TODO()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()

	// Loop through cursor and process items
	for cursor.Next(context.TODO()) {
		var item models.Item
		if err := cursor.Decode(&item); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	if err := cursor.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(items)
}
