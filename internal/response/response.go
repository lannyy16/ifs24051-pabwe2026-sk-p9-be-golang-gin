package response
import "github.com/gin-gonic/gin"
func Success(c *gin.Context, code int, message string, data any) { c.JSON(code,gin.H{"status":"success","message":message,"data":data}) }
func Fail(c *gin.Context, code int, message string) { c.JSON(code,gin.H{"status":"error","message":message}) }
func Error(c *gin.Context, err error) { msg:="Internal server error"; if err!=nil { msg=err.Error() }; Fail(c,500,msg) }