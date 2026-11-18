package config
import "testing"
func TestLoad(t *testing.T){t.Setenv("APP_PORT","");c:=Load();if c.Port!="8081"||c.JWTSecret==""{t.Fatal("defaults")};t.Setenv("APP_PORT","9000");if Load().Port!="9000"{t.Fatal("env")}}