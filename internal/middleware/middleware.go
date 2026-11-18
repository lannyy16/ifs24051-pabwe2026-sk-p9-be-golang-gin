package middleware
import (
 "net/http"
 "strings"
 "time"
 "github.com/gin-gonic/gin"
 "github.com/golang-jwt/jwt/v5"
)
type Claims struct { UserID uint `json:"user_id"`; Role string `json:"role"`; jwt.RegisteredClaims }
func CreateToken(id uint, role, secret string) (string,error) {
 c:=Claims{UserID:id,Role:role,RegisteredClaims:jwt.RegisteredClaims{ExpiresAt:jwt.NewNumericDate(time.Now().Add(24*time.Hour))}}
 return jwt.NewWithClaims(jwt.SigningMethodHS256,c).SignedString([]byte(secret))
}
func Auth(secret string) gin.HandlerFunc { return func(c *gin.Context) {
 parts:=strings.SplitN(c.GetHeader("Authorization")," ",2)
 if len(parts)!=2 || !strings.EqualFold(parts[0],"Bearer") { c.AbortWithStatusJSON(401,gin.H{"status":"error","message":"Unauthorized"}); return }
 token,err:=jwt.ParseWithClaims(parts[1],&Claims{},func(t *jwt.Token)(any,error){ if t.Method!=jwt.SigningMethodHS256{return nil,jwt.ErrSignatureInvalid}; return []byte(secret),nil })
 if err!=nil || !token.Valid { c.AbortWithStatusJSON(401,gin.H{"status":"error","message":"Invalid or expired token"}); return }
 claims,ok:=token.Claims.(*Claims); if !ok { c.AbortWithStatus(401); return }
 c.Set("userID",claims.UserID); c.Set("role",claims.Role); c.Next()
} }
func RequireRole(role string) gin.HandlerFunc { return func(c *gin.Context){ actual,_:=c.Get("role"); if actual!=role { c.AbortWithStatusJSON(403,gin.H{"status":"error","message":"Forbidden"}); return }; c.Next() } }
func CORS(origins string) gin.HandlerFunc { return func(c *gin.Context){
 origin:=c.GetHeader("Origin"); for _,allowed:=range strings.Split(origins,","){ if strings.TrimSpace(allowed)==origin && origin!="" { c.Header("Access-Control-Allow-Origin",origin); c.Header("Vary","Origin") } }
 c.Header("Access-Control-Allow-Headers","Authorization, Content-Type"); c.Header("Access-Control-Allow-Methods","GET, POST, PUT, PATCH, DELETE, OPTIONS")
 if c.Request.Method==http.MethodOptions { c.AbortWithStatus(204); return }; c.Next()
} }