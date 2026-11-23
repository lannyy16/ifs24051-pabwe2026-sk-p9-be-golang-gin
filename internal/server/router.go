// Package server merangkai seluruh rute API, Swagger UI, dan handler 404.
package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"
)

// NewRouter membuat Gin engine lengkap dengan seluruh dependensi.
func NewRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	authH := auth.NewHandler(auth.NewService(auth.NewRepository(db), cfg.AppKey))
	postH := post.NewHandler(post.NewService(post.NewRepository(db)))
	aiH := ai.NewHandler(ai.NewService(
		ai.NewSearchClient(cfg.SearchEngineEndpoint, ""),
		ai.NewChatClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel),
	))

	r.GET("/", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Delcom Blog API berjalan", gin.H{"name": "Delcom Blog API", "version": "1.0.0"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api.json")))
	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		spec, err := docs.GetSwaggerJSON(c.Request.Host, scheme)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal membuat spesifikasi OpenAPI")
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", spec)
	})

	api := r.Group("/api", middleware.RateLimiter(300, time.Minute))

	pub := api.Group("/public")
	pub.GET("/categories", postH.Categories)
	pub.GET("/posts", postH.PublicList)
	pub.GET("/posts/:slug", postH.PublicDetail)

	authG := api.Group("/auth")
	authG.POST("/register", authH.Register)
	authG.POST("/login", authH.Login)
	protectedAuth := authG.Group("", middleware.Auth(cfg.AppKey))
	protectedAuth.POST("/logout", authH.Logout)
	protectedAuth.GET("/profile", authH.Profile)
	protectedAuth.PUT("/profile", authH.UpdateProfile)

	posts := api.Group("/posts", middleware.Auth(cfg.AppKey))
	posts.GET("", postH.List)
	posts.POST("", postH.Create)
	posts.GET("/stats", postH.Stats)
	posts.POST("/ai-generate", aiH.Generate)
	posts.GET("/:id", postH.Detail)
	posts.PUT("/:id", postH.Update)
	posts.PATCH("/:id", postH.Update)
	posts.PATCH("/:id/toggle-publish", postH.TogglePublish)
	posts.DELETE("/:id", postH.Delete)

	admin := api.Group("/admin", middleware.Auth(cfg.AppKey), middleware.RequireRole(auth.RoleAdmin))
	admin.GET("/users", postH.AdminUsers)
	admin.GET("/posts", postH.AdminPosts)
	admin.GET("/users/:id/posts", postH.AdminUserPosts)

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})
	return r
}
