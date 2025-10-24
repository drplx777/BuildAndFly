package main

import (
	"buildANDfly/pkg/config"
	"buildANDfly/pkg/constants"
	"buildANDfly/pkg/controllers"
	"buildANDfly/pkg/database"
	"buildANDfly/pkg/middleware"
	"buildANDfly/pkg/service"
	"context"
	swaglay "github.com/KoNekoD/swaglay/pkg"
	"github.com/KoNekoD/swaglay/pkg/adapters/swaglay_fiber"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	cfg := config.MustLoad()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dbURL := config.BuildDBConnectionString(cfg.DB)
	dbPool, err := database.NewPool(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}
	defer dbPool.Close()

	// init storage (MinIO/S3 compatible)
	storage, err := service.NewMinioStorage(cfg.S3.Endpoint, cfg.S3.AccessKey, cfg.S3.SecretKey, cfg.S3.UseSSL, cfg.S3.Bucket, cfg.S3.BaseURL)
	if err != nil {
		log.Fatalf("storage init error: %v", err)
	}
	//service
	authService := service.NewAuthService(dbPool, cfg.JWTSecret)
	courseService := service.NewCourseService(dbPool, storage)
	mediaService := service.NewMediaService(dbPool, storage)

	//fiber init
	app := fiber.New()

	//swagger init
	swaglay.SetupApi("BuildAndFlyApi")
	swaglay_fiber.Fiber = app
	swaglay_fiber.FiberApp = app

	//healthchek
	app.Get("/health", func(c fiber.Ctx) error {
		if err := dbPool.Ping(context.Background()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).SendString("DB not ready")
		}
		return c.SendString("OK")
	})

	// Middleware
	app.Use(middleware.SlogLogger())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     cfg.CORS.AllowMethods,
		AllowHeaders:     cfg.CORS.AllowHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		ExposeHeaders:    cfg.CORS.ExposeHeaders,
	}))

	//app.Get("/api/getuserbyJWT", authHandler.GetUserHandler)

	// controllers
	authHandler := controllers.NewAuthHandler(authService)
	courseController := controllers.NewCourseController(courseService)
	adminController := controllers.NewAdminController(authService, courseService, storage, dbPool)
	uploadController := controllers.NewUploadController(authService, mediaService)

	//регистрация маршрутов
	authHandler.RegisterRoutes(app)
	courseController.RegisterRoutes(app)
	adminController.RegisterRoutes(app)
	uploadController.RegisterRoutes(app)

	//activate swagger
	includeApiDocs(app)

	//gracefull shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Starting server", "port", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("Server failed", "error", err)
			shutdown <- syscall.SIGTERM
		}
	}()

	<-shutdown
	slog.Info("Shutting down server...")

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}

	slog.Info("Server stopped")
}

func includeApiDocs(r *fiber.App) {
	apiSpec, _ := swaglay.Api.Spec()

	swaggerBytes, err := apiSpec.MarshalJSON()
	if err != nil {
		panic(err)
	}

	// Examples:
	// * Stoplight Elements https://stoplight.io/open-source/elements
	// * Redocly https://redocly.com/

	htmlBytes, err := os.ReadFile("pkg/templates/api-docs2.html")
	if err != nil {
		panic(err)
	}

	jsBytes, err := os.ReadFile("pkg/templates/redoc.standalone.js")
	if err != nil {
		panic(err)
	}
	jstrying, err := os.ReadFile("pkg/templates/try-it-out.js")
	if err != nil {
		panic(err)
	}

	cssBytes, err := os.ReadFile("pkg/templates/swagger-ui.css")
	if err != nil {
		panic(err)
	}

	getSwagger := func(ctx fiber.Ctx) error {
		ctx.Set("Content-Type", constants.JsContentType)
		return ctx.Send(swaggerBytes)
	}

	getDocsHtml := func(ctx fiber.Ctx) error {
		ctx.Set("Content-Type", constants.HtmlContentType)
		return ctx.Send(htmlBytes)
	}

	getDocsJs := func(ctx fiber.Ctx) error {
		ctx.Set("Content-Type", constants.JsContentType)
		return ctx.Send(jsBytes)
	}
	getDocsJs2 := func(ctx fiber.Ctx) error {
		ctx.Set("Content-Type", constants.JsContentType)
		return ctx.Send(jstrying)
	}

	getDocsCss := func(ctx fiber.Ctx) error {
		ctx.Set("Content-Type", constants.CssContentType)
		return ctx.Send(cssBytes)
	}

	r.Group("/api/docs").
		Get("", getDocsHtml).
		Get("/redoc.standalone.js", getDocsJs).
		Get("/swagger-ui.css", getDocsCss).
		Get("/try-it-out.js", getDocsJs2).
		Get("/openapi.json", getSwagger)
}
