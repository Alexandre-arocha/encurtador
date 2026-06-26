package links

import "errors"

// Erros de domínio. Os handlers mapeiam cada um para um status HTTP e uma
// mensagem em PT-BR voltada ao usuário.
var (
	ErrNotFound      = errors.New("link não encontrado")
	ErrSlugTaken     = errors.New("slug já está em uso")
	ErrInvalidSlug   = errors.New("slug inválido")
	ErrReservedSlug  = errors.New("slug reservado")
	ErrInvalidURL    = errors.New("URL de destino inválida")
	ErrSlugGenFailed = errors.New("não foi possível gerar um slug único")
)
