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
	"HamzaMasood1/cooking-app/api/handlers"
	"HamzaMasood1/cooking-app/api/models"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gin-contrib/sessions"
	redisStore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var err error
var recipesHandler *handlers.RecipesHandler
var authHandler *handlers.AuthHandler

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

	// --redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	status := redisClient.Ping()
	fmt.Println(status)

	//--recipes handler
	recipesHandler = handlers.NewRecipesHandler(ctx, collection, redisClient)

	//--jwt handler
	collectionUsers := client.Database(os.Getenv("MONGO_DATABASE")).Collection("users")
	authHandler = handlers.NewAuthHandler(ctx, collectionUsers)
}

func main() {
	router := gin.Default()
	router.GET("/recipes", recipesHandler.ListRecipesHandler)
	router.POST("/signin", authHandler.SignInHandler)
	router.POST("/refresh", authHandler.RefreshHandler)
	authorized := router.Group("/")
	authorized.Use(authHandler.AuthMiddleware())
	{
		authorized.POST("/recipes", recipesHandler.NewRecipeHandler)
		authorized.GET("/recipes/search", recipesHandler.SearchRecipeHandler)
		authorized.GET("/recipes/:id", recipesHandler.GetOneRecipeHandler)
		authorized.PUT("/recipes/:id", recipesHandler.UpdateRecipeHandler)
		authorized.DELETE("recipes/:id", recipesHandler.DeleteRecipeHandler)
	}
	//redis store
	store, _ := redisStore.NewStore(10, "tcp", "localhost:6379", "", []byte("secret"))
	router.Use(sessions.Sessions("recipes_api", store))
	router.Run("localhost:8080")
}
