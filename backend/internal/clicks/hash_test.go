package clicks

import "testing"

func TestHashIPUsesSaltAndIsDeterministic(t *testing.T) {
	const ip = "203.0.113.10"

	first := HashIP(ip, "salt-a")
	second := HashIP(ip, "salt-a")
	otherSalt := HashIP(ip, "salt-b")

	if first == "" {
		t.Fatal("hash vazio")
	}
	if first != second {
		t.Fatalf("hash nao deterministico: %q != %q", first, second)
	}
	if first == ip {
		t.Fatal("IP cru nao pode ser o valor persistido")
	}
	if first == otherSalt {
		t.Fatal("salt diferente deveria gerar hash diferente")
	}
}
