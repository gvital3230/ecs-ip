package web

import (
	"ecs-ip/internal/aws"
	"slices"
	"sort"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
)

type FiberServer struct {
	*fiber.App
}

func NewServer(region string, password string) *FiberServer {
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

	server.Get("/", func(c *fiber.Ctx) error {
		clusters := aws.NewStore(region).Clusters()
		selectedApp := c.Query("app")

		return Render(c, HomePage(filteredByApp(clusters, selectedApp), appSlugs(clusters), selectedApp))
	})

	return server
}

func appSlugs(clusters []aws.Cluster) []string {
	res := []string{}
	for _, cluster := range clusters {
		for _, service := range cluster.Services {
			if service.App != "" {
				if !slices.Contains(res, service.App) {
					res = append(res, service.App)
				}
			}
		}
	}
	sort.Strings(res)
	return res
}

func filteredByApp(clusters []aws.Cluster, app string) []aws.Cluster {
	if app == "" {
		return clusters
	}
	res := []aws.Cluster{}
	for _, cluster := range clusters {
		services := []aws.Service{}
		for _, service := range cluster.Services {
			if service.App == app {
				services = append(services, service)
			}
		}
		if len(services) > 0 {
			cluster.Services = services
			res = append(res, cluster)
		}
	}
	return res
}

// helper function which allows to render templ component and wrap in to fiber handler
func Render(c *fiber.Ctx, component templ.Component, options ...func(*templ.ComponentHandler)) error {
	componentHandler := templ.Handler(component)
	for _, o := range options {
		o(componentHandler)
	}
	return adaptor.HTTPHandler(componentHandler)(c)
}
