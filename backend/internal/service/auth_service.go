package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"qt-devops-go-mongo/backend/internal/apperror"
	"qt-devops-go-mongo/backend/internal/cache"
	"qt-devops-go-mongo/backend/internal/config"
	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/repository"
)

const jwtIssuer = "projecthub-api"

type AuthService interface {
	// Register creates a "member" account and returns a token for it.
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	// Authenticate validates a JWT and loads its user from the database, so deleted users and
	// role changes take effect immediately.
	Authenticate(ctx context.Context, token string) (*models.User, error)
}

type authService struct {
	users   *repository.UserRepository
	cache   cache.Cache
	secret  []byte
	ttl     time.Duration
	userTTL time.Duration
}

type claims struct {
	Role models.Role `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(users *repository.UserRepository, c cache.Cache, cfg config.Config) AuthService {
	return &authService{users: users, cache: c, secret: []byte(cfg.JWTSecret), ttl: cfg.JWTExpiresIn, userTTL: cfg.CacheUserTTL}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		ID:           primitive.NewObjectID(),
		Name:         strings.TrimSpace(req.Name),
		Email:        normalizeEmail(req.Email),
		PasswordHash: string(hash),
		Role:         models.RoleMember,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, apperror.Conflict("email is already registered")
		}
		return nil, err
	}
	return s.issue(user)
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, apperror.Unauthorized("invalid email or password")
	}
	return s.issue(user)
}

func (s *authService) Authenticate(ctx context.Context, token string) (*models.User, error) {
	var cl claims
	_, err := jwt.ParseWithClaims(token, &cl, func(*jwt.Token) (any, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(jwtIssuer),
		// Reject non-canonical base64 (e.g. altered padding bits) so each token has exactly one valid encoding.
		jwt.WithStrictDecoding())
	if err != nil {
		return nil, apperror.Unauthorized("invalid or expired token")
	}
	id, err := primitive.ObjectIDFromHex(cl.Subject)
	if err != nil {
		return nil, apperror.Unauthorized("invalid token subject")
	}

	// Every authenticated request needs the user; cache it briefly. UserService evicts the entry
	// when the role changes or the user is deleted, and the short TTL bounds staleness otherwise.
	var cached models.User
	if s.cache.Get(ctx, cache.UserKey(id), &cached) {
		return &cached, nil
	}
	user, err := s.users.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.Unauthorized("user no longer exists")
	}
	if err != nil {
		return nil, err
	}
	s.cache.Set(ctx, cache.UserKey(id), user, s.userTTL)
	return user, nil
}

func (s *authService) issue(u *models.User) (*dto.AuthResponse, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.Hex(),
			Issuer:    jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	})
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return nil, err
	}
	return &dto.AuthResponse{Token: signed, ExpiresIn: int(s.ttl.Seconds()), User: u.Public()}, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
