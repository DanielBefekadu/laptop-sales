package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RestockLaptop struct {
	ID            primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	LaptopID      primitive.ObjectID `json:"laptopId,omitempty" bson:"laptopId,omitempty"`
	Change        int                `json:"change,omitempty" bson:"change,omitempty"`
	Type          string             `json:"type"`
	PreviousStock int                `json:"previousStock"`
	NewStock      int                `json:"newStock"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
}

type OrderedLaptop struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	LaptopID  primitive.ObjectID `json:"laptopId,omitempty" bson:"laptopId,omitempty"`
	Quantity  int                `json:"quantity,omitempty" bson:"quantity,omitempty"`
	Status    string             `json:"status,omitempty" bson:"status,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

type MongoDataBase struct {
	Db     *mongo.Database
	Client *mongo.Client
}

type Laptop struct {
	ID                primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Brand             string             `json:"brand,omitempty" bson:"brand,omitempty"`
	Model             string             `json:"model,omitempty" bson:"model,omitempty"`
	Price             float64            `json:"price,omitempty" bson:"price,omitempty"`
	CPU               string             `json:"cpu,omitempty" bson:"cpu,omitempty"`
	CPUGen            int                `json:"cpuGen,omitempty" bson:"cpuGen,omitempty"`
	GPU               string             `json:"gpu,omitempty" bson:"gpu,omitempty"`
	RAM               int                `json:"ram,omitempty" bson:"ram,omitempty"`
	StorageCapacity   int                `json:"storagecapacity,omitempty" bson:"storagecapacity,omitempty"`
	StorageType       string             `json:"storagetype,omitempty" bson:"storageType,omitempty"`
	Description       string             `json:"description,omitempty" bson:"description,omitempty"`
	Stock             int                `json:"stock" bson:"stock,omitempty"`
	ReserveStock      int                `json:"reserveStock" bson:"reserveStock,omitempty"`
	LowStockThreshold int                `json:"lowStockThreshold" bson:"lowStockThreshold,omitempty"`
}

type Order struct {
	Message string `json:"message"`
	Item    Laptop `json:"item,omitempty"`
}

var mongoDataBase MongoDataBase

const (
	defaultConnectionString = "mongodb+srv://kach:Wiz*3264@gotestcluster.wgks8f4.mongodb.net/"
	defaultDBName           = "laptopdb"
)

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func connection() {
	connectionString := getEnv("MONGODB_URI", defaultConnectionString)
	dbName := getEnv("MONGODB_DB", defaultDBName)

	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err = client.Connect(ctx)

	if err != nil {
		log.Fatal("Could not connect to MongoDB. Check your firewall or credentials:", err)
	}

	db := client.Database(dbName)

	mongoDataBase = MongoDataBase{
		Client: client,
		Db:     db,
	}

	fmt.Println("Connected to MongoDB!")
	_, err = mongoDataBase.Db.Collection("laptops").Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "brand", Value: 1}}},
		{Keys: bson.D{{Key: "price", Value: 1}}},
		{Keys: bson.D{{Key: "ram", Value: 1}}},
	})

}

func basicAuth(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token != "123456789" {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return c.Next()
}

func FilterLaptops(c *fiber.Ctx) error {
	filter := bson.M{}

	if search := c.Query("search"); search != "" {
		filter["$or"] = []bson.M{
			{"brand": bson.M{"$regex": search, "$options": "i"}},
			{"model": bson.M{"$regex": search, "$options": "i"}},
			{"description": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	if brand := c.Query("brand"); brand != "" {
		filter["brand"] = brand
	}
	if cpu := c.Query("cpu"); cpu != "" {
		filter["cpu"] = cpu
	}
	if gpu := c.Query("gpu"); gpu != "" {
		filter["gpu"] = gpu
	}

	// if ram := c.Query("ram"); ram != "" {
	// 	filter["ram"] = ram
	// }

	priceFilter := bson.M{}
	if minPrice := c.QueryFloat("minPrice"); minPrice > 0 {
		priceFilter["$gte"] = minPrice
	}
	if maxPrice := c.QueryFloat("maxPrice"); maxPrice > 0 {
		priceFilter["$lte"] = maxPrice
	}

	ramFilter := bson.M{}

	if ram := c.QueryInt("ram"); ram > 0 {
		ramFilter["$gte"] = ram
	}

	if len(priceFilter) > 0 {
		filter["price"] = priceFilter
	}
	if len(ramFilter) > 0 {
		filter["ram"] = ramFilter
	}

	filter["stock"] = bson.M{"$gt": 0}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	skip := (page - 1) * limit

	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))

	cursor, err := mongoDataBase.Db.Collection("laptops").Find(c.Context(), filter, opts)
	if err != nil {
		return fmt.Errorf("failed to execute find: %w", err)
	}

	defer cursor.Close(c.Context())

	laptops := make([]Laptop, 0)
	if err := cursor.All(c.Context(), &laptops); err != nil {
		return fmt.Errorf("failed to decode documents: %w", err)
	}
	return c.JSON(laptops)
}

func GetStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	newStatus := c.Query("status")

	if newStatus == "" {
		return fmt.Errorf("status query parameter is required")
	}

	validStatuses := map[string]bool{
		"Pending":   true,
		"Shipped":   true,
		"Delivered": true,
		"Cancelled": true,
	}

	if !validStatuses[newStatus] {
		return fmt.Errorf("invalid status value: %s", newStatus)
	}

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("failed to parse ID: %w", err)
	}

	query := bson.M{"_id": objectId}
	update := bson.M{"$set": bson.M{"status": newStatus}}
	options := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var order OrderedLaptop
	err = mongoDataBase.Db.Collection("orders").FindOneAndUpdate(c.Context(), query, update, options).Decode(&order)

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return c.JSON(order)
}

func GetAvalibaleLaptops(c *fiber.Ctx) error {
	query := bson.M{"stock": bson.M{"$gt": 0}}

	cursor, err := mongoDataBase.Db.Collection("laptops").Find(c.Context(), query)
	if err != nil {
		return fmt.Errorf("failed to execute find: %w", err)
	}
	defer cursor.Close(c.Context())

	laptops := make([]Laptop, 0)
	if err := cursor.All(c.Context(), &laptops); err != nil {
		return fmt.Errorf("failed to decode documents: %w", err)
	}
	return c.JSON(laptops)
}

func LowStock(c *fiber.Ctx) error {
	filter := bson.M{
		"$expr": bson.M{
			"$lte": []interface{}{"$stock", "$lowStockThreshold"},
		},
	}

	cursor, err := mongoDataBase.Db.Collection("laptops").Find(c.Context(), filter)

	if err != nil {
		return fmt.Errorf("failed to execute find: %w", err)
	}
	defer cursor.Close(c.Context())

	laptops := make([]Laptop, 0)
	if err := cursor.All(c.Context(), &laptops); err != nil {
		return fmt.Errorf("failed to decode documents: %w", err)
	}
	return c.JSON(laptops)
}

func OrderLaptop(c *fiber.Ctx) error {
	id := c.Params("id")
	objectId, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return fmt.Errorf("failed to parse ID: %w", err)
	}

	query := bson.M{"_id": objectId, "stock": bson.M{"$gt": 0}}
	update := bson.M{"$inc": bson.M{"stock": -1, "reserveStock": 1}}

	options := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var laptop Laptop
	err = mongoDataBase.Db.Collection("laptops").FindOneAndUpdate(c.Context(), query, update, options).Decode(&laptop)

	if laptop.Stock <= laptop.LowStockThreshold {
		sendLaptopStockAlert(laptop.Stock, laptop.Brand, laptop.Model)
	}

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(Order{Message: "Laptop is out of stock"})
		}
		return fmt.Errorf("failed to update laptop stock: %w", err)
	}

	order := OrderedLaptop{
		LaptopID:  laptop.ID,
		Quantity:  1,
		Status:    "Pending",
		CreatedAt: time.Now(),
	}

	insertResult, err := mongoDataBase.Db.Collection("orders").InsertOne(c.Context(), order)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	order.ID = insertResult.InsertedID.(primitive.ObjectID)
	go func(orderID, laptopID primitive.ObjectID) {
		time.Sleep(3 * time.Minute) // 3-minute reservation
		ctx := context.Background()

		// Check if order is still pending
		var pendingOrder OrderedLaptop
		err := mongoDataBase.Db.Collection("orders").FindOne(ctx, bson.M{"_id": orderID, "status": "Pending"}).Decode(&pendingOrder)
		if err == nil {
			// Release stock
			mongoDataBase.Db.Collection("laptops").UpdateOne(ctx, bson.M{"_id": laptopID}, bson.M{"$inc": bson.M{"stock": 1, "reserveStock": -1}})
			// Update order status
			mongoDataBase.Db.Collection("orders").UpdateOne(ctx, bson.M{"_id": orderID}, bson.M{"$set": bson.M{"status": "Cancelled"}})
		}
	}(order.ID, laptop.ID)

	history := RestockLaptop{
		LaptopID:      laptop.ID,
		Change:        -1,
		Type:          "Order",
		PreviousStock: laptop.Stock + 1,
		NewStock:      laptop.Stock,
		CreatedAt:     time.Now(),
	}

	_, err = mongoDataBase.Db.Collection("restockHistory").InsertOne(c.Context(), history)

	return c.JSON(Order{Message: "Laptop ordered successfully", Item: laptop})
}

func GetAllOrders(c *fiber.Ctx) error {
	query := bson.D{{}}
	cursor, err := mongoDataBase.Db.Collection("orders").Find(c.Context(), query)
	if err != nil {
		return fmt.Errorf("failed to execute find: %w", err)
	}
	var orders []OrderedLaptop
	defer cursor.Close(c.Context())
	cursor.All(c.Context(), &orders)

	return c.JSON(orders)
}

func sendLaptopStockAlert(stock int, brand string, model string) {
	fmt.Printf("ALERT: Laptop %s %s stock is low! Only %d left in stock.\n", brand, model, stock)
}

func getPclist(c *fiber.Ctx) error {

	query := bson.D{{}}

	cursour, err := mongoDataBase.Db.Collection("laptops").Find(c.Context(), query)
	if err != nil {
		return fmt.Errorf("failed to execute find: %w", err)
	}

	defer cursour.Close(c.Context())

	laptops := make([]Laptop, 0)

	if err := cursour.All(c.Context(), &laptops); err != nil {
		return fmt.Errorf("failed to decode documents: %w", err)
	}

	return c.JSON(laptops)
}

func postPclist(c *fiber.Ctx) error {
	laptop := new(Laptop)
	if err := c.BodyParser(laptop); err != nil {
		return fmt.Errorf("failed to parse body: %w", err)
	}

	if laptop.Brand == "" || laptop.Model == "" {
		return fmt.Errorf("brand and model are required fields")
	}

	if laptop.Price < 0 {
		return fmt.Errorf("price cannot be negative")
	}

	if laptop.RAM < 0 {
		return fmt.Errorf("RAM cannot be negative")
	}

	if laptop.StorageCapacity < 0 {
		return fmt.Errorf("storage capacity cannot be negative")
	}

	if laptop.Stock < 0 {
		return fmt.Errorf("stock cannot be negative")
	}

	if laptop.LowStockThreshold < 0 {
		return fmt.Errorf("low stock threshold cannot be negative")
	}

	allLaptop, err := mongoDataBase.Db.Collection("laptops").InsertOne(c.Context(), laptop)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	laptop.ID = allLaptop.InsertedID.(primitive.ObjectID)

	history := RestockLaptop{
		LaptopID:      laptop.ID,
		Change:        laptop.Stock,
		Type:          "Initial Stock",
		PreviousStock: 0,
		NewStock:      laptop.Stock,
		CreatedAt:     time.Now(),
	}

	_, err = mongoDataBase.Db.Collection("restockHistory").InsertOne(c.Context(), history)
	if err != nil {
		return fmt.Errorf("failed to insert restock history: %w", err)
	}
	return c.JSON(laptop)
}

func getPclistById(c *fiber.Ctx) error {

	id := c.Params("id")
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("failed to parse ID: %w", err)
	}
	query := bson.M{"_id": objectId}

	var laptop Laptop

	err = mongoDataBase.Db.Collection("laptops").FindOne(c.Context(), query).Decode(&laptop)
	if err != nil {
		return fmt.Errorf("failed to decode document: %w", err)
	}

	return c.JSON(laptop)
}

func deletePclist(c *fiber.Ctx) error {
	id := c.Params("id")
	objectId, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return fmt.Errorf("failed to parse ID: %w", err)
	}

	query := bson.M{"_id": objectId}

	deltedPc := mongoDataBase.Db.Collection("laptops").FindOneAndDelete(c.Context(), query)

	var laptop Laptop

	err = deltedPc.Decode(&laptop)
	if err != nil {
		return fmt.Errorf("failed to decode document: %w", err)
	}
	return c.JSON(laptop)
}

func putPclist(c *fiber.Ctx) error {
	id := c.Params("id")
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("failed to parse ID: %w", err)
	}

	query := bson.M{"_id": objectId}
	var laptop Laptop

	if err := c.BodyParser(&laptop); err != nil {
		return fmt.Errorf("failed to parse body: %w", err)
	}

	// update := bson.M{
	// 	"$set": bson.M{
	// 		"brand":           laptop.Brand,
	// 		"model":           laptop.Model,
	// 		"price":           laptop.Price,
	// 		"cpu":             laptop.CPU,
	// 		"cpuGen":          laptop.CPUGen,
	// 		"gpu":             laptop.GPU,
	// 		"ram":             laptop.RAM,
	// 		"storagecapacity": laptop.StorageCapacity,
	// 		"storageType":     laptop.StorageType,
	// 		"description":     laptop.Description,
	// 	},
	// }

	update := bson.M{}

	if laptop.Brand != "" {
		update["brand"] = laptop.Brand
	}
	if laptop.Model != "" {
		update["model"] = laptop.Model
	}
	if laptop.Price != 0 {
		update["price"] = laptop.Price
	}

	if laptop.CPU != "" {
		update["cpu"] = laptop.CPU
	}
	if laptop.CPUGen != 0 {
		update["cpuGen"] = laptop.CPUGen
	}
	if laptop.GPU != "" {
		update["gpu"] = laptop.GPU
	}
	if laptop.RAM != 0 {
		update["ram"] = laptop.RAM
	}
	if laptop.StorageCapacity != 0 {
		update["storagecapacity"] = laptop.StorageCapacity
	}
	if laptop.StorageType != "" {
		update["storageType"] = laptop.StorageType
	}
	if laptop.Description != "" {
		update["description"] = laptop.Description
	}
	if laptop.Stock >= 0 { // includes 0
		update["stock"] = laptop.Stock
	}
	if laptop.LowStockThreshold != 0 {
		update["lowStockThreshold"] = laptop.LowStockThreshold
	}
	updateData := bson.M{"$set": update}

	options := options.FindOneAndUpdate().SetReturnDocument(options.After)
	updatePc := mongoDataBase.Db.Collection("laptops").FindOneAndUpdate(c.Context(), query, updateData, options)

	var updatedLaptop Laptop

	err = updatePc.Decode(&updatedLaptop)
	if err != nil {
		return fmt.Errorf("failed to decode document: %w", err)
	}

	return c.JSON(updatedLaptop)
}

func main() {

	connection()
	app := fiber.New()
	port := getEnv("PORT", "5000")

	app.Get("/pclist", getPclist)
	app.Post("/pclist", basicAuth, postPclist)
	app.Delete("/pclist/:id", basicAuth, deletePclist)
	app.Put("/pclist/:id", basicAuth, putPclist)
	app.Get("/pclist/:id", getPclistById)
	app.Get("/availablelaptops", GetAvalibaleLaptops)
	app.Post("/orderlaptop/:id", basicAuth, OrderLaptop)
	app.Get("/shop", FilterLaptops)
	app.Get("/orders", basicAuth, GetAllOrders)
	app.Get("/low-stock", basicAuth, LowStock)
	app.Patch("/orders/:id/status", basicAuth, GetStatus)

	log.Printf("Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
