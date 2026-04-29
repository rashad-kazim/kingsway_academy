package auth

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"kingsway/backend/internal/domain"
)

type UserStore interface {
	CountOwners(ctx context.Context) (int, error)
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUser(ctx context.Context, id string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
}

type Service struct {
	store     UserStore
	jwtSecret string
}

type BootstrapOwnerInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type CreateUserInput struct {
	BranchID  string      `json:"branch_id"`
	Role      domain.Role `json:"role"`
	Email     string      `json:"email"`
	Password  string      `json:"password"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func NewService(store UserStore, jwtSecret string) *Service {
	return &Service{store: store, jwtSecret: jwtSecret}
}

func (s *Service) BootstrapOwner(ctx context.Context, input BootstrapOwnerInput) (LoginResult, error) {
	ownerCount, err := s.store.CountOwners(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	if ownerCount > 0 {
		return LoginResult{}, domain.ErrConflict
	}

	user, err := s.CreateUser(ctx, domain.Principal{Role: domain.RoleOwner}, CreateUserInput{
		Role:      domain.RoleOwner,
		Email:     input.Email,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		return LoginResult{}, err
	}

	token, err := SignToken(s.jwtSecret, Claims{
		UserID:   user.ID,
		BranchID: user.BranchID,
		Role:     user.Role,
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Token: token, User: user}, nil
}

func (s *Service) CreateUser(ctx context.Context, actor domain.Principal, input CreateUserInput) (domain.User, error) {
	if input.Password == "" || input.Email == "" || input.FirstName == "" || input.LastName == "" || !input.Role.IsValid() {
		return domain.User{}, domain.ErrInvalidInput
	}
	if input.Role == domain.RoleOwner && actor.Role != domain.RoleOwner {
		return domain.User{}, domain.ErrForbidden
	}
	if input.Role != domain.RoleOwner {
		if !actor.IsOwner() && actor.Role != domain.RoleReceptionist {
			return domain.User{}, domain.ErrForbidden
		}
		if !actor.IsOwner() {
			input.BranchID = actor.BranchID
		}
		if input.BranchID == "" {
			return domain.User{}, domain.ErrInvalidInput
		}
	}

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return domain.User{}, err
	}

	return s.store.CreateUser(ctx, domain.User{
		BranchID:     input.BranchID,
		Role:         input.Role,
		Email:        strings.TrimSpace(input.Email),
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(input.FirstName),
		LastName:     strings.TrimSpace(input.LastName),
	})
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	user, err := s.store.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return LoginResult{}, domain.ErrUnauthorized
	}
	if !user.IsActive || !CheckPassword(user.PasswordHash, input.Password) {
		return LoginResult{}, domain.ErrUnauthorized
	}

	token, err := SignToken(s.jwtSecret, Claims{
		UserID:   user.ID,
		BranchID: user.BranchID,
		Role:     user.Role,
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Token: token, User: user}, nil
}

func (s *Service) PrincipalFromToken(raw string) (domain.Principal, error) {
	claims, err := ParseToken(s.jwtSecret, raw)
	if err != nil {
		return domain.Principal{}, err
	}

	return domain.Principal{
		UserID:   claims.UserID,
		BranchID: claims.BranchID,
		Role:     claims.Role,
	}, nil
}

func (s *Service) CurrentUser(ctx context.Context, principal domain.Principal) (domain.User, error) {
	if principal.UserID == "" {
		return domain.User{}, domain.ErrUnauthorized
	}

	user, err := s.store.GetUser(ctx, principal.UserID)
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	if !user.IsActive {
		return domain.User{}, domain.ErrUnauthorized
	}

	return user, nil
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", domain.ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func RequireAnyRole(principal domain.Principal, roles ...domain.Role) error {
	for _, role := range roles {
		if principal.Role == role {
			return nil
		}
	}

	return domain.ErrForbidden
}

func RequireBranch(principal domain.Principal, branchID string) error {
	if branchID == "" {
		return domain.ErrInvalidInput
	}
	if principal.CanUseBranch(branchID) {
		return nil
	}

	return domain.ErrForbidden
}

func IsAuthError(err error) bool {
	return errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, ErrInvalidToken)
}
