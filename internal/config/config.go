package config
import "os"
type Config struct { Port, DatabaseURL, JWTSecret, FrontendOrigins, OpenAIKey, OpenAIEndpoint, OpenAIModel, SearchEndpoint string }
func value(k,d string) string { if v:=os.Getenv(k); v!="" { return v }; return d }
func Load() Config { return Config{
 Port:value("APP_PORT","8081"),
 DatabaseURL:value("DATABASE_URL","postgres://postgres:postgres@localhost:5432/delcom_blog?sslmode=disable"),
 JWTSecret:value("JWT_SECRET","development-secret"),
 FrontendOrigins:value("FRONTEND_ORIGINS","http://localhost:5174"),
 OpenAIKey:os.Getenv("OPENAI_API_KEY"),
 OpenAIEndpoint:value("OPENAI_ENDPOINT","https://api.openai.com/v1/chat/completions"),
 OpenAIModel:value("OPENAI_MODEL","gpt-4o-mini"),
 SearchEndpoint:os.Getenv("SEARCH_ENGINE_ENDPOINT"),
} }