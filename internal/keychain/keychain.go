package keychain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func Store(service, account, token string) error {
	cmd := exec.Command("/usr/bin/security", "add-generic-password", "-U", "-s", service, "-a", account, "-w", token)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("store token in Keychain: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func Load(service, account string) (string, error) {
	cmd := exec.Command("/usr/bin/security", "find-generic-password", "-s", service, "-a", account, "-w")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read token from Keychain: %w", err)
	}
	token := strings.TrimSpace(string(output))
	if len(token) < 43 {
		return "", fmt.Errorf("Keychain token is unexpectedly short")
	}
	return token, nil
}
