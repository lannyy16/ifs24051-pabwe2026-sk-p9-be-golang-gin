package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"app/internal/middleware"
)

// Error domain autentikasi.
var (
	ErrUsernameTaken      = errors.New("username sudah digunakan")
	ErrEmailTaken         = errors.New("email sudah digunakan")
	ErrInvalidCredentials = errors.New("username atau password salah")
	ErrUserNotFound       = errors.New("pengguna tidak ditemukan")
)

// signToken dapat diganti pada pengujian untuk mensimulasikan kegagalan penandatanganan.
var signToken = func(t *jwt.Token, key []byte) (string, error) { return t.SignedString(key) }

// Service berisi logika bisnis autentikasi.
type Service struct {
	repo   Repository
	secret string
	ttl    time.Duration
	admins map[string]bool // username akun admin bawaan (huruf kecil)
}

// NewService membuat Service dengan secret JWT (APP_KEY).
func NewService(repo Repository, secret string) *Service {
	return &Service{repo: repo, secret: secret, ttl: 24 * time.Hour}
}

// WithAdmins menandai username akun admin bawaan (mis. "admin" dan ADMIN_USERNAME).
// Username ini dicadangkan: tidak dapat didaftarkan lewat Register, dan bila akun
// bawaan ternyata tersimpan dengan role selain admin (data lama), role-nya otomatis
// dipulihkan saat login sehingga endpoint /api/admin/* tidak menolak admin (403).
func (s *Service) WithAdmins(usernames ...string) *Service {
	s.admins = map[string]bool{}
	for _, u := range usernames {
		if u = strings.ToLower(strings.TrimSpace(u)); u != "" {
			s.admins[u] = true
		}
	}
	return s
}

func (s *Service) isReservedAdmin(username string) bool {
	return s.admins[strings.ToLower(strings.TrimSpace(username))]
}

// Register mendaftarkan penulis baru (role member) dan langsung menerbitkan token.
func (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if s.isReservedAdmin(username) {
		return nil, ErrUsernameTaken
	}
	taken, err := s.repo.UsernameExists(username, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrUsernameTaken
	}
	taken, err = s.repo.EmailExists(email, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrEmailTaken
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &User{Username: username, Email: email, Password: string(hash), Role: RoleMember}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return s.authResponse(user)
}

// Login memverifikasi kredensial dan menerbitkan token JWT.
func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByIdentity(strings.TrimSpace(req.Username))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCredentials
	}
	if s.isReservedAdmin(user.Username) && user.Role != RoleAdmin {
		user.Role = RoleAdmin
		if err := s.repo.Update(user); err != nil {
			return nil, err
		}
	}
	return s.authResponse(user)
}

// Profile mengambil profil pengguna berdasarkan ID.
func (s *Service) Profile(id uint) (*UserResponse, error) {
	user, err := s.find(id)
	if err != nil {
		return nil, err
	}
	res := ToResponse(user)
	return &res, nil
}

// UpdateProfile memperbarui username, email, dan/atau password.
func (s *Service) UpdateProfile(id uint, req UpdateProfileRequest) (*UserResponse, error) {
	user, err := s.find(id)
	if err != nil {
		return nil, err
	}
	if username := strings.TrimSpace(req.Username); username != "" && username != user.Username {
		taken, err := s.repo.UsernameExists(username, id)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrUsernameTaken
		}
		user.Username = username
	}
	if email := strings.ToLower(strings.TrimSpace(req.Email)); email != "" && email != user.Email {
		taken, err := s.repo.EmailExists(email, id)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrEmailTaken
		}
		user.Email = email
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hash)
	}
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	res := ToResponse(user)
	return &res, nil
}

func (s *Service) find(id uint) (*User, error) {
	user, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return user, err
}

func (s *Service) authResponse(user *User) (*AuthResponse, error) {
	claims := middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := signToken(jwt.NewWithClaims(jwt.SigningMethodHS256, claims), []byte(s.secret))
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: ToResponse(user)}, nil
}
