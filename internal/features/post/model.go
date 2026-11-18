package post
import "time"
type Post struct {
 ID uint `json:"id" gorm:"primaryKey"`
 UserID uint `json:"user_id" gorm:"index;not null"`
 Title string `json:"title" gorm:"not null"`
 Slug string `json:"slug" gorm:"uniqueIndex;not null"`
 Category string `json:"category" gorm:"not null"`
 Excerpt string `json:"excerpt"`
 Content string `json:"content" gorm:"not null"`
 IsPublished bool `json:"is_published"`
 PublishedAt *time.Time `json:"published_at"`
 ReadingTime int `json:"reading_time"`
 CreatedAt time.Time `json:"created_at"`
 UpdatedAt time.Time `json:"updated_at"`
}