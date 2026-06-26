package links

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

const (
	// base62Alphabet é o conjunto usado na geração de slugs automáticos.
	base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// generatedSlugLength é o tamanho do slug gerado automaticamente.
	generatedSlugLength = 7
	// maxSlugAttempts limita as tentativas de gerar um slug livre de colisão.
	maxSlugAttempts = 6
)

// slugFormat valida o formato de slugs customizados: letras, dígitos, hífen e
// underscore, de 1 a 64 caracteres.
var slugFormat = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// reservedSlugs não podem ser usados como slug porque colidem com rotas fixas.
var reservedSlugs = map[string]struct{}{
	"api":     {},
	"healthz": {},
}

// ValidateSlugFormat valida um slug customizado fornecido pelo usuário.
func ValidateSlugFormat(slug string) error {
	if !slugFormat.MatchString(slug) {
		return fmt.Errorf("%w: use apenas letras, números, hífen ou underscore (1 a 64 caracteres)", ErrInvalidSlug)
	}
	if _, reserved := reservedSlugs[strings.ToLower(slug)]; reserved {
		return fmt.Errorf("%w: %q não pode ser usado", ErrReservedSlug, slug)
	}
	return nil
}

// generateSlug gera um slug base62 aleatório (crypto/rand, sem viés).
func generateSlug() (string, error) {
	b := make([]byte, generatedSlugLength)
	max := big.NewInt(int64(len(base62Alphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("erro ao gerar slug aleatório: %w", err)
		}
		b[i] = base62Alphabet[n.Int64()]
	}
	return string(b), nil
}
