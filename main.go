// package main

// import (
// 	"fmt"
// 	"log"
// 	"os"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/joho/godotenv"
// )

// type Todo struct{
// 	ID int `json:"id"`
// 	Completed bool `json:"completed"`
// 	Body string `json:"body"`
// }

// func main() {
// 	// fmt.Println("Hello worlds")

// 	app := fiber.New()

// 	err := godotenv.Load(".env")

// 	if err != nil {
// 		log.Fatal("Error loading .env file")
// 	}

// 	PORT := os.Getenv("PORT")

// 	todos := []Todo{}

// 	app.Get("/api/todos", func(c *fiber.Ctx) error{
// 		// return c.Status(200).JSON(fiber.Map{"msg":"Hello world"})
// 		return c.Status(200).JSON(todos)
// 	})

// 	// Create a todo
// 	app.Post("/api/todos", func(c *fiber.Ctx) error {
// 		todo := &Todo{}  // {id:0, completed: false, body: ""}
// 		if err := c.BodyParser(todo); err != nil {
// 			return err;
// 		}

// 		if todo.Body == ""{
// 			return c.Status(400).JSON(fiber.Map{"error":"Todo body is required"})
// 		}

// 		todo.ID = len(todos)+1
// 		todos = append(todos, *todo)
// 		// fmt.Println(todos)
// 		return c.Status(200).JSON(todo)
// 	})

// 	// Update a todo
// 	app.Patch("/api/todos/:id", func(c *fiber.Ctx) error {
// 		id := c.Params("id")

// 		for i, todo := range todos{
// 			if fmt.Sprint(todo.ID) == id {
// 				todos[i].Completed = true;
// 				return c.Status(200).JSON(todos[i])
// 			}
// 		}

// 		return c.Status(400).JSON(fiber.Map{"error":"Todo not found"})
// 	})

// 	// Delete a todo
// 	app.Delete("/api/todos/:id", func(c *fiber.Ctx) error {
// 		id := c.Params("id")

// 		for i, todo := range todos{
// 			if fmt.Sprint(todo.ID) == id {
// 				// todos[i].Completed = true;
// 				todos = append(todos[:i], todos[i+1:]...)
// 				return c.Status(200).JSON(fiber.Map{"todos":todos,"success":true})
// 			}
// 		}

// 		return c.Status(404).JSON(fiber.Map{"error":"Todo not found"})
// 	})

// 	log.Fatal(app.Listen(`:`+ PORT))
// }

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	// "github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


type Todo struct{
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Completed bool `json:"completed"`
	Body string `json:"body"`
}

var collection *mongo.Collection


func main(){
	// fmt.Println("Hello world")

	if os.Getenv("ENV") != "production"{
		// Load the .env file if not in production
		err := godotenv.Load(".env")
		if(err != nil){
			log.Fatal("Error loading .env file: ", err)
		}
	}

	MONGODB_URI := os.Getenv("MONGODB_URI")

	clientOptions := options.Client().ApplyURI(MONGODB_URI)

	client, err := mongo.Connect(context.Background(), clientOptions) 

	if err != nil{
		log.Fatal(err)
	}

	defer client.Disconnect(context.Background())

	err = client.Ping(context.Background(), nil)

	if err != nil{
		log.Fatal(err)
	}

	fmt.Println("Connected to Mongodb Atlas")

	collection = client.Database("golang_db").Collection("todos")

	app := fiber.New()

	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: os.Getenv("FRONT_URL"),
	// 	AllowHeaders: "Origin,Content-Type,Accept",
	// }))

	if os.Getenv("ENV") == "production"{
		app.Static("/","./client/dist")
	}

	app.Get("/api/todos", getTodos)
	app.Post("/api/todos", createTodos)
	app.Patch("/api/todos/:id", updateTodos)
	app.Delete("/api/todos/:id", deleteTodos)

	port := os.Getenv("PORT")

	if port == ""{
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))

	
}

func getTodos(c *fiber.Ctx) error{
	var todos []Todo 
	
	cursor, err := collection.Find(context.Background(), bson.M{})

	if(err != nil){
		return err;
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()){
		var todo Todo
		if err := cursor.Decode(&todo); err != nil{
			return err;
		}

		todos = append(todos, todo)
	}
	return c.JSON(todos)
}

func createTodos(c *fiber.Ctx) error{
	
	newTodo := new(Todo)
	if err := c.BodyParser(newTodo); err != nil{
		return err;
	}

	if newTodo.Body == ""{
		return c.Status(400).JSON(fiber.Map{"error":"Todo body can't be empty"})
	}
	
	cursor, err := collection.Find(context.Background(), bson.M{})
	
	if(err != nil){
		return err;
	}
	for cursor.Next(context.Background()){
		var todo Todo
		if err := cursor.Decode(&todo); err != nil{
			return err;
		}
		if strings.EqualFold(strings.ReplaceAll(newTodo.Body, " ", ""), strings.ReplaceAll(todo.Body, " ", "")) {
			return c.Status(401).JSON(fiber.Map{"error":"Todo already present"})
		}

	}
	
	insertResult, err := collection.InsertOne(context.Background(), newTodo)

	if err != nil{
		return err;
	}

	newTodo.ID = insertResult.InsertedID.(primitive.ObjectID)

	return c.Status(201).JSON(newTodo)
}

func updateTodos(c *fiber.Ctx) error{
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)

	if err != nil{
		return c.Status(400).JSON(fiber.Map{"error":"Invalid todo ID"});
	}
	
	filter := bson.M{"_id": objectID}

	update := bson.M{"$set": bson.M{"completed": true}}

	_,err = collection.UpdateOne(context.Background(), filter, update)
	
	if err != nil{
		return err;
	}
	return c.Status(200).JSON(fiber.Map{"success":true});
}

func deleteTodos(c *fiber.Ctx) error{
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)

	if err != nil{
		return c.Status(400).JSON(fiber.Map{"error":"Invalid todo ID"});
	}
	
	filter := bson.M{"_id": objectID}

	// update := bson.M{"$set": bson.M{"completed": true}}

	_,err = collection.DeleteOne(context.Background(), filter)
	
	if err != nil{
		return err;
	}
	return c.Status(200).JSON(fiber.Map{"success":true});
}
