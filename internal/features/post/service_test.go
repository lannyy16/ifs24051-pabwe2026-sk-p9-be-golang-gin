package post
import("strings";"testing")
func TestPostHelpers(t *testing.T){
 if Slugify(" Belajar Go & Gin! ")!="belajar-go-gin"{t.Fatal("slug")}
 if Slugify("---")!=""{t.Fatal("empty slug")}
 if ReadingTime("")!=1||ReadingTime(strings.Repeat("kata ",201))!=2{t.Fatal("reading time")}
 if !Valid("Judul","Teknologi","Isi")||Valid(" ","Teknologi","Isi"){t.Fatal("validation")}
 if Category(" teknologi ")!="Teknologi"||Category("")!="Umum"{t.Fatal("category")}
}