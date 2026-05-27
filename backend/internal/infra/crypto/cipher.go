package crypto

// Cipher 适配应用层的 SecretCipher 端口。
type Cipher struct{}

func (Cipher) Encrypt(plaintext string) (string, error) {
	return Encrypt(plaintext)
}

func (Cipher) Decrypt(ciphertext string) (string, error) {
	return Decrypt(ciphertext)
}
