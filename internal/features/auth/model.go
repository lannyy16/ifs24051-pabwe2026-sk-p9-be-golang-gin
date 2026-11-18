package auth
import "time"
type User struct {
 ID uint `json:"id" gorm:"primaryKey"`
 Username string `json:"username" gorm:"uniqueIndex;not null"`
 Email string `json:"email" gorm:"uniqueIndex;not null"`
 Password string `json:"-" gorm:"not null"`
 Role string `json:"role" gorm:"default:author"`
 CreatedAt time.Time `json:"created_at"`
}