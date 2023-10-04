package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Book struct represents the book details
type Book struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Name   string             `bson:"name"`
	Author string             `bson:"author"`
	Cost   float64            `bson:"cost"`
}

// MongoDB configuration
const (
	mongoURI     = "mongodb://localhost:27017"
	dbName       = "bookstore"
	collection   = "books"
	mongoTimeout = 5 * time.Second
)

var tpl *template.Template
var client *mongo.Client

func init() {
	// Initialize MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), mongoTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)
	c, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	client = c

	// Load templates
	tpl = template.Must(template.ParseFiles("index.html"))
}

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/submit", handleSubmit)
	http.HandleFunc("/delete", handleDelete) // Register delete handler before starting the server
	http.HandleFunc("/modify", handleModify) // Register modify handler
	fmt.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	books, err := getBooks()
	if err != nil {
		http.Error(w, "Failed to get books", http.StatusInternalServerError)
		return
	}

	data := struct {
		Books []Book
	}{
		Books: books,
	}

	tpl.Execute(w, data)
}

func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve form values
	name := r.FormValue("name")
	author := r.FormValue("author")
	cost := r.FormValue("cost")

	// Convert cost to float64
	bookCost, err := strconv.ParseFloat(cost, 64)
	if err != nil {
		http.Error(w, "Invalid cost", http.StatusBadRequest)
		return
	}

	// Create a Book instance
	book := Book{
		ID:     primitive.NewObjectID(),
		Name:   name,
		Author: author,
		Cost:   bookCost,
	}

	// Insert book into MongoDB
	coll := client.Database(dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoTimeout)
	defer cancel()

	_, err = coll.InsertOne(ctx, book)
	if err != nil {
		http.Error(w, "Failed to insert book", http.StatusInternalServerError)
		return
	}

	// Display success message using JavaScript alert
	fmt.Fprintf(w, `<script>alert('Book added successfully!'); window.location.href = '/';</script>`)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bookID := r.FormValue("id")
	if bookID == "" {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	objID, err := primitive.ObjectIDFromHex(bookID)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	coll := client.Database(dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoTimeout)
	defer cancel()

	result, err := coll.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		http.Error(w, "Failed to delete book", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		// Display error message using JavaScript alert
		fmt.Fprintf(w, `<script>alert('Book not found!'); window.location.href = '/';</script>`)
		return
	}

	// Display success message using JavaScript alert
	fmt.Fprintf(w, `<script>alert('Book deleted successfully!'); window.location.href = '/';</script>`)
}

func handleModify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bookID := r.FormValue("id")
	if bookID == "" {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	objID, err := primitive.ObjectIDFromHex(bookID)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	// Retrieve form values
	name := r.FormValue("name")
	author := r.FormValue("author")
	cost := r.FormValue("cost")

	// Convert cost to float64
	bookCost, err := strconv.ParseFloat(cost, 64)
	if err != nil {
		http.Error(w, "Invalid cost", http.StatusBadRequest)
		return
	}

	// Update book details in the database
	coll := client.Database(dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoTimeout)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name":   name,
			"author": author,
			"cost":   bookCost,
		},
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		http.Error(w, "Failed to update book", http.StatusInternalServerError)
		return
	}

	// Display success message using JavaScript alert
	fmt.Fprintf(w, `<script>alert('Book modified successfully!'); window.location.href = '/';</script>`)
}

func getBooks() ([]Book, error) {
	coll := client.Database(dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoTimeout)
	defer cancel()

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var books []Book
	for cursor.Next(ctx) {
		var book Book
		if err := cursor.Decode(&book); err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return books, nil
}
