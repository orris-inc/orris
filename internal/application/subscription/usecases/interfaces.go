package usecases

type TokenGenerator interface {
	Generate(prefix string) (plainToken string, hash string, err error)
}
