package post
import (
 "regexp"
 "strings"
 "unicode"
)
var invalidSlug=regexp.MustCompile(`[^a-z0-9]+`)
func Slugify(s string) string { return strings.Trim(invalidSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)),"-"),"-") }
func ReadingTime(s string) int { n:=(len(strings.Fields(s))+199)/200;if n<1{return 1};return n }
func Valid(title,category,content string) bool { return strings.TrimSpace(title)!=""&&strings.TrimSpace(category)!=""&&strings.TrimSpace(content)!="" }
func Category(s string) string { s=strings.TrimSpace(s);if s==""{return "Umum"};r:=[]rune(s);r[0]=unicode.ToUpper(r[0]);return string(r) }