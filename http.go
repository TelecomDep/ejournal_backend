package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type AuthBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func startHTTPServer() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000,http://127.0.0.1:3000",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)
	app.Get("/profile", profileHandler)

	log.Println("Starting HTTP server on :8888")
	if err := app.Listen(":8888"); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func registerHandler(c *fiber.Ctx) error {
	var body AuthBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling request"})
	}

	req := Request{ID: "http-register", Action: "register", Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := dispatchRequest(string(raw), 3*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

func loginHandler(c *fiber.Ctx) error {
	var body AuthBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling request"})
	}

	req := Request{ID: "http-login", Action: "login", Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := dispatchRequest(string(raw), 3*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		return c.Status(fiber.StatusUnauthorized).JSON(resp)
	}

	return c.JSON(resp)
}

func profileHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(Response{OK: false, Error: "missing Authorization header"})
	}

	req := Request{ID: "http-profile", Action: "profile", Token: token}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := dispatchRequest(string(raw), 3*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		return c.Status(fiber.StatusUnauthorized).JSON(resp)
	}

	return c.JSON(resp)
}
