package controllers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"golang-api/config"
	"golang-api/controllers"
	"golang-api/models"
)

type MongoClient interface {
    Database(name string) MongoDatabase
}

type MongoDatabase interface {
    Collection(name string) MongoCollection
}

type MongoCollection interface {
    InsertOne(ctx context.Context, document interface{},
        opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
    FindOne(ctx context.Context, filter interface{},
        opts ...*options.FindOneOptions) *mongo.SingleResult
    UpdateOne(ctx context.Context, filter interface{}, update interface{},
        opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
    DeleteOne(ctx context.Context, filter interface{},
        opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
}

type MockMongoClient struct {
    mock.Mock
}

func (m *MockMongoClient) Database(name string) MongoDatabase {
    args := m.Called(name)
    return args.Get(0).(MongoDatabase)
}

type MockDatabase struct {
    mock.Mock
}

func (m *MockDatabase) Collection(name string) MongoCollection {
    args := m.Called(name)
    return args.Get(0).(MongoCollection)
}

type MockCollection struct {
    mock.Mock
}

func (m *MockCollection) InsertOne(ctx context.Context, document interface{},
    opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
    args := m.Called(ctx, document)
    return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockCollection) FindOne(ctx context.Context, filter interface{},
    opts ...*options.FindOneOptions) *mongo.SingleResult {
    args := m.Called(ctx, filter)
    return args.Get(0).(*mongo.SingleResult)
}

func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{},
    opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
    args := m.Called(ctx, filter, update)
    return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockCollection) DeleteOne(ctx context.Context, filter interface{},
    opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
    args := m.Called(ctx, filter)
    return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

type MockRedisClient struct {
    mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
    args := m.Called(ctx, key)
    return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
    args := m.Called(ctx, key, value, expiration)
    return args.Get(0).(*redis.StatusCmd)
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
    args := m.Called(ctx, keys)
    return args.Get(0).(*redis.IntCmd)
}

func TestCreateItem(t *testing.T) {
    mockMongo := new(MockMongoClient)
    mockDB := new(MockDatabase)
    mockColl := new(MockCollection)

    mockMongo.On("Database", "testdb").Return(mockDB)
    mockDB.On("Collection", "items").Return(mockColl)

    item := models.Item{
        Name:        "Test Item",
        Description: "This is a test item",
        Price:       99.99,
    }
    payload, _ := json.Marshal(item)
    req, _ := http.NewRequest("POST", "/items", strings.NewReader(string(payload)))
    rr := httptest.NewRecorder()

    mockColl.On("InsertOne", config.Ctx, mock.Anything).Return(&mongo.InsertOneResult{
        InsertedID: primitive.NewObjectID(),
    }, nil)

    http.HandlerFunc(controllers.CreateItem).ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "status code not OK")
    var createdItem models.Item
    err := json.Unmarshal(rr.Body.Bytes(), &createdItem)
    assert.Nil(t, err, "error unmarshalling response")
    assert.Equal(t, item.Name, createdItem.Name)
    assert.Equal(t, item.Description, createdItem.Description)
    assert.Equal(t, item.Price, createdItem.Price)
    mockMongo.AssertExpectations(t)
    mockDB.AssertExpectations(t)
    mockColl.AssertExpectations(t)
}

func TestGetItem(t *testing.T) {
    mockMongo := new(MockMongoClient)
    mockDB := new(MockDatabase)
    mockColl := new(MockCollection)
    mockRedis := new(MockRedisClient)

    mockMongo.On("Database", "testdb").Return(mockDB)
    mockDB.On("Collection", "items").Return(mockColl)

    itemID := primitive.NewObjectID()
    req, _ := http.NewRequest("GET", "/items/"+itemID.Hex(), nil)
    rr := httptest.NewRecorder()

    mockRedis.On("Get", config.Ctx, itemID.Hex()).Return(redis.NewStringResult("", redis.Nil))

    mockColl.On("FindOne", config.Ctx, bson.M{"_id": itemID}).Return(&mongo.SingleResult{})

    router := mux.NewRouter()
    router.HandleFunc("/items/{id}", controllers.GetItem).Methods("GET")
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusInternalServerError, rr.Code, "status code should be internal server error as no data found")
    mockMongo.AssertExpectations(t)
    mockDB.AssertExpectations(t)
    mockColl.AssertExpectations(t)
    mockRedis.AssertExpectations(t)
}

func TestUpdateItem(t *testing.T) {
    mockMongo := new(MockMongoClient)
    mockDB := new(MockDatabase)
    mockColl := new(MockCollection)

    // Mock setup for MongoDB client calls
    mockMongo.On("Database", "testdb").Return(mockDB)
    mockDB.On("Collection", "items").Return(mockColl)

    itemID := primitive.NewObjectID()
    updatedItem := models.Item{
        Name:        "Updated Test Item",
        Description: "This is an updated test item",
        Price:       149.99,
    }
    payload, _ := json.Marshal(updatedItem)
    req, _ := http.NewRequest("PUT", "/items/"+itemID.Hex(), strings.NewReader(string(payload)))
    rr := httptest.NewRecorder()

    mockColl.On("UpdateOne", config.Ctx, bson.M{"_id": itemID}, bson.M{"$set": updatedItem}).Return(
        &mongo.UpdateResult{ModifiedCount: 1}, nil)

    router := mux.NewRouter()
    router.HandleFunc("/items/{id}", controllers.UpdateItem).Methods("PUT")
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "status code not OK")
    var returnedItem models.Item
    err := json.Unmarshal(rr.Body.Bytes(), &returnedItem)
    assert.Nil(t, err, "error unmarshalling response")
    assert.Equal(t, updatedItem.Name, returnedItem.Name)
    assert.Equal(t, updatedItem.Description, returnedItem.Description)
    assert.Equal(t, updatedItem.Price, returnedItem.Price)
    mockMongo.AssertExpectations(t)
    mockDB.AssertExpectations(t)
    mockColl.AssertExpectations(t)
}

func TestDeleteItem(t *testing.T) {
    mockMongo := new(MockMongoClient)
    mockDB := new(MockDatabase)
    mockColl := new(MockCollection)

    mockMongo.On("Database", "testdb").Return(mockDB)
    mockDB.On("Collection", "items").Return(mockColl)

    itemID := primitive.NewObjectID()
    req, _ := http.NewRequest("DELETE", "/items/"+itemID.Hex(), nil)
    rr := httptest.NewRecorder()

    mockColl.On("DeleteOne", config.Ctx, bson.M{"_id": itemID}).Return(
        &mongo.DeleteResult{DeletedCount: 1}, nil)

    router := mux.NewRouter()
    router.HandleFunc("/items/{id}", controllers.DeleteItem).Methods("DELETE")
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "status code not OK")
    var responseMessage map[string]string
    err := json.Unmarshal(rr.Body.Bytes(), &responseMessage)
    assert.Nil(t, err, "error unmarshalling response")
    assert.Equal(t, "Deleted", responseMessage["message"])
    mockMongo.AssertExpectations(t)
    mockDB.AssertExpectations(t)
    mockColl.AssertExpectations(t)
}


func TestGetAllItems(t *testing.T) {
    mockMongo := new(MockMongoClient)
    mockDB := new(MockDatabase)
    mockColl := new(MockCollection)

    mockMongo.On("Database", "testdb").Return(mockDB)
    mockDB.On("Collection", "items").Return(mockColl)

    req, _ := http.NewRequest("GET", "/items", nil)
    rr := httptest.NewRecorder()

    mockCursor := &mongo.Cursor{}
    mockColl.On("Find", context.TODO(), bson.M{}, &options.FindOptions{}).Return(mockCursor, nil)

    router := mux.NewRouter()
    router.HandleFunc("/items", controllers.GetAllItems).Methods("GET")
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "status code not OK")
    var returnedItems []models.Item
    err := json.Unmarshal(rr.Body.Bytes(), &returnedItems)
    assert.Nil(t, err, "error unmarshalling response")
    mockMongo.AssertExpectations(t)
    mockDB.AssertExpectations(t)
    mockColl.AssertExpectations(t)
}