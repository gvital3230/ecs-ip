package main

import (
	"ecs-ip/internal/web"
	"fmt"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("ADMIN_PASSWORD is not set")
	}

	region := os.Getenv("REGION")
	server := web.NewServer(region, password)

	host := os.Getenv("HOST")
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	err := server.Listen(fmt.Sprintf("%v:%d", host, port))
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
