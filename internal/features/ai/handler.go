package ai
import(
 "bytes"
 "encoding/json"
 "net/http"
 "strings"
 "time"
 "github.com/gin-gonic/gin"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/config"
 "ifs24051-pabwe2026-sk-p9-be-golang-gin/internal/response"
)
type Handler struct{Config config.Config}
type input struct{Topic string `json:"topic"`;Category string `json:"category"`;IncludeWebSearch bool `json:"include_web_search"`}
type completion struct{Model string `json:"model"`;Messages []map[string]string `json:"messages"`;ResponseFormat map[string]string `json:"response_format"`}
type result struct{Choices []struct{Message struct{Content string `json:"content"`} `json:"message"`} `json:"choices"`}
func(h Handler) Generate(c *gin.Context){
 var in input;if c.ShouldBindJSON(&in)!=nil||strings.TrimSpace(in.Topic)==""{response.Fail(c,400,"Topik wajib diisi");return}
 if h.Config.OpenAIKey==""{response.Fail(c,503,"OPENAI_API_KEY belum dikonfigurasi");return}
 prompt:="Buat artikel blog bahasa Indonesia. Balas JSON dengan title, category, excerpt, content_markdown. Topik: "+in.Topic+". Kategori: "+in.Category
 payload,_:=json.Marshal(completion{Model:h.Config.OpenAIModel,Messages:[]map[string]string{{"role":"system","content":"Anda editor blog profesional."},{"role":"user","content":prompt}},ResponseFormat:map[string]string{"type":"json_object"}})
 req,err:=http.NewRequestWithContext(c.Request.Context(),"POST",h.Config.OpenAIEndpoint,bytes.NewReader(payload));if err!=nil{response.Error(c,err);return}
 req.Header.Set("Authorization","Bearer "+h.Config.OpenAIKey);req.Header.Set("Content-Type","application/json")
 client:=http.Client{Timeout:60*time.Second};res,err:=client.Do(req);if err!=nil{response.Fail(c,502,"Gagal menghubungi layanan AI");return};defer res.Body.Close()
 if res.StatusCode<200||res.StatusCode>=300{response.Fail(c,502,"Layanan AI mengembalikan respons gagal");return}
 var out result;if json.NewDecoder(res.Body).Decode(&out)!=nil||len(out.Choices)==0{response.Fail(c,502,"Format respons AI tidak valid");return}
 var article map[string]any;if json.Unmarshal([]byte(out.Choices[0].Message.Content),&article)!=nil{response.Fail(c,502,"AI tidak mengembalikan JSON artikel");return}
 response.Success(c,200,"Draf berhasil dibuat",gin.H{"article":article,"sources":[]any{}})
}