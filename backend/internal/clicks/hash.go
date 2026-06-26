package clicks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HashIP gera o valor persistido em clicks.ip_hash. O IP cru nunca deve ser
// gravado em banco, logs ou job args.
func HashIP(rawIP, salt string) string {
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write([]byte(strings.TrimSpace(rawIP)))
	return hex.EncodeToString(mac.Sum(nil))
}
