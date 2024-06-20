package models

import (
	"time"
)

type Item struct {
    ID          string             `json:"id" bson:"_id"`
    Name        string             `json:"name"`
    Description string             `json:"description"`
    Price       float64            `json:"price"`
    CreatedAt   time.Time          `json:"created_at"`
    UpdatedAt   time.Time          `json:"updated_at"`
}
