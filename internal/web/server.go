package web

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
)

type FiberServer struct {
	*fiber.App
}

func NewServer() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "ecs-ip",
			AppName:      "ecs-ip",
		}),
	}

	server.Use(cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.Query("noCache") == "true"
		},
		Expiration:   5 * time.Minute,
		CacheControl: true,
	}))

	return server
}
