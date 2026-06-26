package links

import (
	"errors"
	"testing"
)

func TestValidateSlugFormat(t *testing.T) {
	tests := []struct {
		name string
		slug string
		want error
	}{
		{name: "valid", slug: "Meu_slug-123"},
		{name: "empty", slug: "", want: ErrInvalidSlug},
		{name: "spaces", slug: "meu slug", want: ErrInvalidSlug},
		{name: "too long", slug: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789___", want: ErrInvalidSlug},
		{name: "api reserved", slug: "api", want: ErrReservedSlug},
		{name: "healthz reserved uppercase", slug: "HEALTHZ", want: ErrReservedSlug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlugFormat(tt.slug)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("ValidateSlugFormat(%q) = %v, want nil", tt.slug, err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("ValidateSlugFormat(%q) = %v, want %v", tt.slug, err, tt.want)
			}
		})
	}
}

func TestGenerateSlugUsesExpectedShape(t *testing.T) {
	slug, err := generateSlug()
	if err != nil {
		t.Fatal(err)
	}
	if len(slug) != generatedSlugLength {
		t.Fatalf("len(slug) = %d, want %d", len(slug), generatedSlugLength)
	}
	if err := ValidateSlugFormat(slug); err != nil {
		t.Fatalf("generated slug invalid: %v", err)
	}
}
