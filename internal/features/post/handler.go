package post
import (
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/gin-gonic/gin"
 "gorm.io/gorm"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/response"
)
type Handler struct{DB *gorm.DB}
type postInput struct{Title string `json:"title"`;Category string `json:"category"`;Excerpt string `json:"excerpt"`;Content string `json:"content"`}
func uid(c *gin.Context) uint{id,_:=c.Get("userID");v,_:=id.(uint);return v}
func(h Handler) Categories(c *gin.Context){response.Success(c,200,"Kategori",[]string{"Teknologi","Pendidikan","Gaya Hidup","Bisnis","Opini","Umum"})}
func(h Handler) PublicList(c *gin.Context){var rows []Post;q:=h.DB.Where("is_published = ?",true);if s:=strings.TrimSpace(c.Query("q"));s!=""{like:="%"+s+"%";q=q.Where("title ILIKE ? OR excerpt ILIKE ?",like,like)};if cat:=c.Query("category");cat!=""&&cat!="Semua"{q=q.Where("category = ?",cat)};if err:=q.Order("published_at DESC").Find(&rows).Error;err!=nil{response.Error(c,err);return};response.Success(c,200,"Artikel publik",rows)}
func(h Handler) PublicDetail(c *gin.Context){var p Post;if h.DB.Where("slug = ? AND is_published = ?",c.Param("slug"),true).First(&p).Error!=nil{response.Fail(c,404,"Artikel tidak ditemukan");return};response.Success(c,200,"Detail artikel",p)}
func(h Handler) List(c *gin.Context){var rows []Post;if err:=h.DB.Where("user_id = ?",uid(c)).Order("updated_at DESC").Find(&rows).Error;err!=nil{response.Error(c,err);return};response.Success(c,200,"Artikel penulis",rows)}
func(h Handler) Create(c *gin.Context){var in postInput;if c.ShouldBindJSON(&in)!=nil||!Valid(in.Title,in.Category,in.Content){response.Fail(c,400,"Judul, kategori, dan konten wajib diisi");return};base:=Slugify(in.Title);slug:=base;for n:=2;;n++{var count int64;h.DB.Model(&Post{}).Where("slug = ?",slug).Count(&count);if count==0{break};slug=base+"-"+strconv.Itoa(n)};p:=Post{UserID:uid(c),Title:strings.TrimSpace(in.Title),Slug:slug,Category:Category(in.Category),Excerpt:in.Excerpt,Content:in.Content,ReadingTime:ReadingTime(in.Content)};if err:=h.DB.Create(&p).Error;err!=nil{response.Error(c,err);return};response.Success(c,201,"Artikel berhasil dibuat",p)}
func(h Handler) Get(c *gin.Context){var p Post;if h.DB.Where("id = ? AND user_id = ?",c.Param("id"),uid(c)).First(&p).Error!=nil{response.Fail(c,404,"Artikel tidak ditemukan");return};response.Success(c,200,"Detail artikel",p)}
func(h Handler) Update(c *gin.Context){var p Post;if h.DB.Where("id = ? AND user_id = ?",c.Param("id"),uid(c)).First(&p).Error!=nil{response.Fail(c,404,"Artikel tidak ditemukan");return};var in postInput;if c.ShouldBindJSON(&in)!=nil||!Valid(in.Title,in.Category,in.Content){response.Fail(c,400,"Judul, kategori, dan konten wajib diisi");return};p.Title=in.Title;p.Category=Category(in.Category);p.Excerpt=in.Excerpt;p.Content=in.Content;p.ReadingTime=ReadingTime(in.Content);if err:=h.DB.Save(&p).Error;err!=nil{response.Error(c,err);return};response.Success(c,200,"Artikel diperbarui",p)}
func(h Handler) Toggle(c *gin.Context){var p Post;if h.DB.Where("id = ? AND user_id = ?",c.Param("id"),uid(c)).First(&p).Error!=nil{response.Fail(c,404,"Artikel tidak ditemukan");return};p.IsPublished=!p.IsPublished;if p.IsPublished{now:=time.Now();p.PublishedAt=&now}else{p.PublishedAt=nil};if err:=h.DB.Save(&p).Error;err!=nil{response.Error(c,err);return};response.Success(c,200,"Status publikasi diperbarui",p)}
func(h Handler) Delete(c *gin.Context){res:=h.DB.Where("id = ? AND user_id = ?",c.Param("id"),uid(c)).Delete(&Post{});if res.Error!=nil{response.Error(c,res.Error);return};if res.RowsAffected==0{response.Fail(c,404,"Artikel tidak ditemukan");return};response.Success(c,200,"Artikel dihapus",nil)}
func(h Handler) Stats(c *gin.Context){var total,pub int64;h.DB.Model(&Post{}).Where("user_id = ?",uid(c)).Count(&total);h.DB.Model(&Post{}).Where("user_id = ? AND is_published = ?",uid(c),true).Count(&pub);response.Success(c,200,"Statistik",gin.H{"total":total,"published":pub,"draft":total-pub})}
func(h Handler) AdminUsers(c *gin.Context){var rows []map[string]any;err:=h.DB.Raw("SELECT users.id,users.username,users.email,users.role,COUNT(posts.id) total_posts,SUM(CASE WHEN posts.is_published THEN 1 ELSE 0 END) total_published FROM users LEFT JOIN posts ON posts.user_id=users.id GROUP BY users.id").Scan(&rows).Error;if err!=nil{response.Error(c,err);return};response.Success(c,200,"Monitoring pengguna",rows)}
func(h Handler) AdminPosts(c *gin.Context){var rows []Post;if err:=h.DB.Order("created_at DESC").Find(&rows).Error;err!=nil{response.Error(c,err);return};response.Success(c,200,"Arsip artikel",rows)}
func(h Handler) AdminUserPosts(c *gin.Context){var rows []Post;if err:=h.DB.Where("user_id = ?",c.Param("id")).Find(&rows).Error;err!=nil{response.Error(c,err);return};response.Success(c,200,"Artikel pengguna",rows)}
var _ = http.StatusOK