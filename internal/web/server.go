package web

import (
	"ecs-ip/internal/aws"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cache"
)

type FiberServer struct {
	*fiber.App
}

func NewServer(password string) *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "ecs-ip",
			AppName:      "ecs-ip",
		}),
	}

	// use basic auth with only one user and password from env
	server.Use(basicauth.New(basicauth.Config{
		Users: map[string]string{
			"admin": password,
		},
	}))

	// cache results for 5 minutes
	server.Use(cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.Query("noCache") == "true"
		},
		Expiration:   5 * time.Minute,
		CacheControl: true,
	}))

	server.Get("/", func(c *fiber.Ctx) error {
		res := aws.NewStore().Clusters()
		return Render(c, HomePage(res))
	})

	return server
}

// helper function which allows to render templ component and wrap in to fiber handler
func Render(c *fiber.Ctx, component templ.Component, options ...func(*templ.ComponentHandler)) error {
	componentHandler := templ.Handler(component)
	for _, o := range options {
		o(componentHandler)
	}
	return adaptor.HTTPHandler(componentHandler)(c)
}
