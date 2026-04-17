package order

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func OrderLaptop(c fiber.Ctx) error {

	var orderRequest struct {
		Items []OrderItem `json:"items"`
	}

	if err := c.BodyParser(&orderRequest); err != nil {
		return fmt.Errorf("failed to parse order request: %v", err)
	}

	if len(orderRequest.Items) == 0 {
		return fmt.Errorf("order must contain at least one item")
	}

	var totalAmount float64


	for i, item := range orderRequest.Items {
		var laptop Laptop
		err := mongoDataBase.Db.Collection("laptops").FindOne(ctx, bson.M{"_id": item.LaptopID}).Decode(&laptop)
		if err != nil {
			return fmt.Errorf("laptop with ID %s not found: %v", item.LaptopID.Hex(), err)
		}
		if laptop.Stock < item.Quantity {
			return fmt.Errorf("not enough stock for laptop %s", laptop.Name)
		}
	}

	}
