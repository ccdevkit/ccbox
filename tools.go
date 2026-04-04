//go:build tools

package tools

import (
	_ "github.com/aymanbagabas/go-pty"
	_ "github.com/ccdevkit/common/settings"
	_ "github.com/creack/pty"
	_ "github.com/google/uuid"
	_ "golang.design/x/clipboard"
	_ "golang.org/x/image/webp"
	_ "golang.org/x/term"
)
