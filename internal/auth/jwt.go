package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

type JWTManager struct {
	secret     []byte
	expiry     time.Duration
	refreshTTL time.Duration
}

type Claims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, expiry, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		expiry:     expiry,
		refreshTTL: refreshTTL,
	}
}

func (j *JWTManager) GenerateTokenPair(userID uuid.UUID) (string, string, int64, error) {
	now := time.Now().UTC()

	accessClaims := Claims{
		UserID:    userID.String(),
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiry)),
		},
	}

	refreshClaims := Claims{
		UserID:    userID.String(),
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTTL)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(j.secret)
	if err != nil {
		return "", "", 0, err
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(j.secret)
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, int64(j.expiry.Seconds()), nil
}

func (j *JWTManager) ParseToken(tokenString string) (uuid.UUID, string, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return uuid.Nil, "", errors.New("empty token")
	}

	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return uuid.Nil, "", err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return uuid.Nil, "", errors.New("invalid token")
	}

	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, "", err
	}

	return uid, claims.TokenType, nil
}

func (j *JWTManager) ParseAccessToken(tokenString string) (uuid.UUID, error) {
	uid, tokenType, err := j.ParseToken(tokenString)
	if err != nil {
		return uuid.Nil, err
	}
	if tokenType != "access" {
		return uuid.Nil, errors.New("token is not an access token")
	}
	return uid, nil
}

func ContextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	uid, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return uid, ok
}

func RequireAuth(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimSpace(authHeader[7:])
			userID, err := jwtManager.ParseAccessToken(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithUserID(r.Context(), userID)))
		})
	}
}
