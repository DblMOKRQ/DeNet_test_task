package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// Ошибки JWT
var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrExpiredToken  = errors.New("token expired")
	ErrInvalidClaims = errors.New("invalid token claims")
)

// Claims представляет данные, хранящиеся в JWT токене
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Service предоставляет методы для работы с JWT
type Service struct {
	secretKey     string
	tokenDuration time.Duration
	log           *zap.Logger
}

// NewService создает новый экземпляр JWT сервиса
func NewService(secretKey string, tokenDuration time.Duration, log *zap.Logger) *Service {

	return &Service{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
		log:           log.Named("jwt_service"),
	}
}

// GenerateToken создает новый JWT токен для пользователя
func (s *Service) GenerateToken(userID string) (string, error) {
	s.log.Debug("Generating token", zap.String("user_id", userID))

	now := time.Now()
	expiresAt := now.Add(s.tokenDuration)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		s.log.Error("Failed to sign token",
			zap.String("user_id", userID),
			zap.Error(err))
		return "", err
	}

	s.log.Info("Token generated successfully",
		zap.String("user_id", userID),
		zap.Time("expires_at", expiresAt))
	return tokenString, nil
}

// ValidateToken проверяет JWT токен и возвращает claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	s.log.Debug("Validating token")

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Проверка алгоритма подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				s.log.Warn("Unexpected signing method",
					zap.String("method", token.Method.Alg()))
				return nil, ErrInvalidToken
			}
			return []byte(s.secretKey), nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			s.log.Warn("Token expired")
			return nil, ErrExpiredToken
		}
		s.log.Warn("Failed to parse token", zap.Error(err))
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		s.log.Warn("Invalid token claims")
		return nil, ErrInvalidClaims
	}

	s.log.Debug("Token validated successfully", zap.String("user_id", claims.UserID))
	return claims, nil
}
