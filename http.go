package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type AuthBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

func startHTTPServer() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: appConfig.CORSAllowOrigins,
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)
	app.Get("/profile", profileHandler)

	app.Post("/api/teacher/attendance-link", teacherAttendanceLinkHandler)
	app.Post("/api/student/attendance/confirm", studentAttendanceConfirmHandler)

	addr := fmt.Sprintf(":%s", appConfig.AppPort)
	log.Printf("Starting HTTP server on %s", addr)
	if err := app.Listen(addr); err != nil {
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

func teacherAttendanceLinkHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(Response{OK: false, Error: "missing Authorization header"})
	}

	var body AttendanceCreateData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling request"})
	}

	req := Request{ID: "http-attendance-link", Action: "create_attendance_link", Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := dispatchRequest(string(raw), 3*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		if resp.Error == "forbidden: teacher role required" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

func studentAttendanceConfirmHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(Response{OK: false, Error: "missing Authorization header"})
	}

	var body AttendanceConfirmData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling request"})
	}

	req := Request{ID: "http-attendance-confirm", Action: "confirm_attendance", Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := dispatchRequest(string(raw), 3*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		if resp.Error == "forbidden: student role required" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		if resp.Error == "attendance already confirmed" {
			return c.Status(fiber.StatusConflict).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}
