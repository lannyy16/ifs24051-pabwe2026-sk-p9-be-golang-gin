package server
import(
 "net/http"
 "github.com/gin-gonic/gin"
 "gorm.io/gorm"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/config"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/features/ai"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/features/auth"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/features/post"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/middleware"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/response"
)
func NewRouter(db *gorm.DB,cfg config.Config)*gin.Engine{
 r:=gin.New();r.Use(gin.Logger(),gin.Recovery(),middleware.CORS(cfg.FrontendOrigins))
 a:=auth.Handler{DB:db,Secret:cfg.JWTSecret};p:=post.Handler{DB:db};aih:=ai.Handler{Config:cfg}
 r.GET("/health",func(c *gin.Context){response.Success(c,200,"API aktif",gin.H{"service":"Delcom Blog"})})
 r.GET("/swagger/*any",func(c *gin.Context){c.Header("Content-Type","text/html");c.String(200,`<html><body><h1>Delcom Blog API</h1><p>API JSON: <a href="/api.json">/api.json</a></p></body></html>`)})
 r.GET("/api.json",func(c *gin.Context){response.Success(c,200,"OpenAPI",gin.H{"openapi":"3.0.3","info":gin.H{"title":"Delcom Blog API","version":"1.0.0"}})})
 api:=r.Group("/api")
 api.GET("/public/categories",p.Categories);api.GET("/public/posts",p.PublicList);api.GET("/public/posts/:slug",p.PublicDetail)
 api.POST("/auth/register",a.Register);api.POST("/auth/login",a.Login)
 secured:=api.Group("");secured.Use(middleware.Auth(cfg.JWTSecret))
 secured.POST("/auth/logout",func(c *gin.Context){response.Success(c,200,"Logout berhasil; hapus token di client",nil)})
 secured.GET("/auth/profile",a.Profile);secured.PUT("/auth/profile",a.UpdateProfile)
 secured.GET("/posts",p.List);secured.POST("/posts",p.Create);secured.GET("/posts/stats",p.Stats);secured.POST("/posts/ai-generate",aih.Generate)
 secured.GET("/posts/:id",p.Get);secured.PUT("/posts/:id",p.Update);secured.PATCH("/posts/:id/toggle-publish",p.Toggle);secured.DELETE("/posts/:id",p.Delete)
 admin:=secured.Group("/admin");admin.Use(middleware.RequireRole("admin"));admin.GET("/users",p.AdminUsers);admin.GET("/posts",p.AdminPosts);admin.GET("/users/:id/posts",p.AdminUserPosts)
 r.NoRoute(func(c *gin.Context){response.Fail(c,http.StatusNotFound,"Endpoint not found")})
 return r
}