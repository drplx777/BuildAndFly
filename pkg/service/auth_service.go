package service

import (
	"buildANDfly/pkg/dtos"
	"context"
	"errors"
	"strings"
	"time"

	"buildANDfly/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// токен живёт неделю
const tokenTTL = 7 * 24 * time.Hour

type AuthService struct {
	dbPool    *pgxpool.Pool
	jwtSecret string
}

func NewAuthService(dbPool *pgxpool.Pool, jwtSecret string) *AuthService {
	return &AuthService{
		dbPool:    dbPool,
		jwtSecret: jwtSecret,
	}
}

var (
	ErrUserExists         = errors.New("user with given email or login already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

func (s *AuthService) getOrCreateRoleID(ctx context.Context, roleName string) (uuid.UUID, error) {
	roleName = strings.ToLower(strings.TrimSpace(roleName))
	if roleName == "" {
		roleName = "user"
	}

	var id uuid.UUID
	err := s.dbPool.QueryRow(ctx, `SELECT id FROM roles WHERE role = $1`, roleName).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	newID := uuid.New()
	_, err = s.dbPool.Exec(ctx, `INSERT INTO roles (id, role) VALUES ($1, $2) ON CONFLICT (role) DO NOTHING`, newID, roleName)
	if err != nil {
		var tried uuid.UUID
		if qerr := s.dbPool.QueryRow(ctx, `SELECT id FROM roles WHERE role = $1`, roleName).Scan(&tried); qerr == nil {
			return tried, nil
		}
		return uuid.Nil, err
	}

	if err := s.dbPool.QueryRow(ctx, `SELECT id FROM roles WHERE role = $1`, roleName).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *AuthService) Register(ctx context.Context, req dtos.Registerdtos) (*models.User, error) {
	if strings.TrimSpace(req.Email) == "" && strings.TrimSpace(req.Login) == "" {
		return nil, errors.New("email or login required")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password too short")
	}

	// проверим существование пользователя
	var exists bool
	if err := s.dbPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 OR login = $2)`, req.Email, req.Login).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	// роль по умолчанию
	//бубу
	roleName := "user"
	if strings.TrimSpace(req.Role) != "" {
		roleName = strings.ToLower(strings.TrimSpace(req.Role))
	}

	roleID, err := s.getOrCreateRoleID(ctx, roleName)
	if err != nil {
		return nil, err
	}

	// хешируем пароль
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// вставляем пользователя
	userID := uuid.New()
	const insertQ = `
		INSERT INTO users (id, name, surname, email, login, password, role_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7, now(), now())
		RETURNING id, name, surname, email, login, role_id, created_at, updated_at
	`
	var user models.User
	if err := s.dbPool.QueryRow(ctx, insertQ,
		userID,
		req.Name,
		req.Surname,
		req.Email,
		req.Login,
		string(hashed),
		roleID,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Surname,
		&user.Email,
		&user.Login,
		&user.RoleID,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	// очищаем пароль в возвращаемой структуре
	user.Password = ""
	return &user, nil
}

// Login аутентифицирует по login или email и возвращает JWT + user + expiry.
func (s *AuthService) Login(ctx context.Context, loginOrEmail, password string) (string, *models.User, time.Time, error) {
	const q = `
		SELECT u.id, u.name, u.surname, u.email, u.login, u.password, u.role_id, r.role
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.login = $1 OR u.email = $1
		LIMIT 1
	`

	var (
		user       models.User
		hashedPass string
		roleID     uuid.UUID
		roleName   string
	)
	if err := s.dbPool.QueryRow(ctx, q, loginOrEmail).Scan(
		&user.ID,
		&user.Name,
		&user.Surname,
		&user.Email,
		&user.Login,
		&hashedPass,
		&roleID,
		&roleName,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, time.Time{}, ErrInvalidCredentials
		}
		return "", nil, time.Time{}, err
	}

	// сравниваем пароль
	if bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(password)) != nil {
		return "", nil, time.Time{}, ErrInvalidCredentials
	}

	// формируем token (токен живёт неделю)
	exp := time.Now().Add(tokenTTL)
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"role":  roleName,
		"iat":   time.Now().Unix(),
		"exp":   exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ts, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", nil, time.Time{}, err
	}

	user.Password = ""
	user.RoleID = roleID
	return ts, &user, exp, nil
}

// ValidateToken парсит и валидирует JWT, возвращает claims.
func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// можно дополнительно проверять алгоритм:
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
func (s *AuthService) SetUserRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	roleID, err := s.getOrCreateRoleID(ctx, roleName)
	if err != nil {
		return err
	}
	_, err = s.dbPool.Exec(ctx, `UPDATE users SET role_id = $1, updated_at = now() WHERE id = $2`, roleID, userID)
	return err
}
func (s *AuthService) GetUserByID(ctx context.Context, userID int) (*models.User, error) {
	const query = `
        SELECT id, name, surname, email, login, roleID, created_at, updated_at
        FROM users WHERE id = $1
    `

	var user models.User
	err := s.dbPool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Name,
		&user.Surname,
		&user.Email,
		&user.Login,
		&user.RoleID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return &user, err
}
