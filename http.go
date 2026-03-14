package main

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
)

type AuthBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func startHTTPServer() {
	app := fiber.New()
	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)
	app.Get("/profile", profileHandler)
	log.Println("Starting HTTP server...")
	app.Listen(":8888")
}

func registerHandler(c *fiber.Ctx) error {
	var body AuthBody

	err := c.BodyParser(&body)
	if err != nil {
		return c.JSON(Response{
			OK:    false,
			Error: "Error parsing body",
		})
	}

	data, _ := json.Marshal(body)

	req := Request{
		ID:     "http-register",
		Action: "register",
		Data:   data,
	}

	raw, _ := json.Marshal(req)

	resp := handleRequest(string(raw))
	return c.JSON(resp)

}

func loginHandler(c *fiber.Ctx) error {
	var body AuthBody
	err := c.BodyParser(&body)
	if err != nil {
		return c.JSON(Response{OK: false, Error: "Error parsing body"})
	}
	data, _ := json.Marshal(body)
	req := Request{
		ID:     "http-login",
		Action: "login",
		Data:   data,
	}
	raw, _ := json.Marshal(req)
	resp := handleRequest(string(raw))
	return c.JSON(resp)
}

func profileHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token != "" {
		return c.Status(fiber.StatusUnauthorized).JSON(Response{
			OK:    false,
			Error: "Invalid token",
		})
	}
	req := Request{
		ID:     "http-profile",
		Action: "profile",
		Token:  token,
	}
	raw, _ := json.Marshal(req)
	resp := handleRequest(string(raw))
	if !resp.OK {
		return c.Status(fiber.StatusUnauthorized).JSON(resp)
	}

	return c.JSON(resp)
}
