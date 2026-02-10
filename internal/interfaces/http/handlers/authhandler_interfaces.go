package handlers

import (
	"context"

	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/domain/user"
)

// Use case interfaces for AuthHandler - enables unit testing with mocks.

type registerUseCase interface {
	Execute(ctx context.Context, cmd usecases.RegisterWithPasswordCommand) (*user.User, error)
}

type loginUseCase interface {
	Execute(ctx context.Context, cmd usecases.LoginWithPasswordCommand) (*usecases.LoginWithPasswordResult, error)
}

type verifyEmailUseCase interface {
	Execute(ctx context.Context, cmd usecases.VerifyEmailCommand) error
}

type requestPasswordResetUseCase interface {
	Execute(ctx context.Context, cmd usecases.RequestPasswordResetCommand) error
}

type resetPasswordUseCase interface {
	Execute(ctx context.Context, cmd usecases.ResetPasswordCommand) error
}

type initiateOAuthUseCase interface {
	Execute(cmd usecases.InitiateOAuthLoginCommand) (*usecases.InitiateOAuthLoginResult, error)
}

type handleOAuthCallbackUseCase interface {
	Execute(ctx context.Context, cmd usecases.HandleOAuthCallbackCommand) (*usecases.HandleOAuthCallbackResult, error)
}

type refreshTokenUseCase interface {
	Execute(ctx context.Context, cmd usecases.RefreshTokenCommand) (*usecases.RefreshTokenResult, error)
}

type logoutUseCase interface {
	Execute(cmd usecases.LogoutCommand) error
}
