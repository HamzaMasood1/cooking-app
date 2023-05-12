// Recipes API
//
//		Schemes: http
//	 Host: localhost:8080
//		BasePath: /
//		Version: 1.0.0
//
//		Consumes:
//		- application/json
//
//		Produces:
//		- application/json
//
// swagger:meta
package main

import (
	"HamzaMasood1/cooking-app/handlers"
	"HamzaMasood1/cooking-app/models"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var err error
var recipesHandler *handlers.RecipesHandler

func init() {
	var client *mongo.Client
	var ctx context.Context
	// --connecting to the database
	file, _ := ioutil.ReadFile("recipes.json")
	recipes := make([]models.Recipe, 0)
	_ = json.Unmarshal(file, &recipes)
	ctx = context.Background()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to MongoDB")

	// --populating the database
	// var listOfRecipes []interface{}
	// for _, recipe := range recipes {
	// 	listOfRecipes = append(listOfRecipes, recipe)
	// }
	// collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	// insertManyResult, err := collection.InsertMany(ctx, listOfRecipes)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println("Inserted recipes: ", insertManyResult.InsertedIDs)

	// --other setup
	collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	recipesHandler = handlers.NewRecipesHandler(ctx, collection)
}

func main() {
	router := gin.Default()
	router.POST("/recipes", recipesHandler.NewRecipeHandler)
	router.GET("/recipes", recipesHandler.ListRecipesHandler)
	router.GET("/recipes/search", recipesHandler.SearchRecipeHandler)
	router.GET("/recipes/:id", recipesHandler.GetOneRecipeHandler)
	router.PUT("/recipes/:id", recipesHandler.UpdateRecipeHandler)
	router.DELETE("recipes/:id", recipesHandler.DeleteRecipeHandler)
	router.Run("localhost:8080")
}
