package main

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	users := map[string]string{
		"admin": "password1",
		"hamza": "password1",
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatal(err)
	}

	collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("users")
	// h := sha256.New()

	for username, password := range users {
		bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 14)
		collection.InsertOne(ctx, bson.M{
			"username": username,
			"password": bytes,
		})
	}
}
