package embed

import (
	"embed"
	"io/fs"
)

//go:embed all:dotfiles
var dotfilesFS embed.FS

// GetDotfilesFS returns the embedded dotfiles filesystem
func GetDotfilesFS() (fs.FS, error) {
	return fs.Sub(dotfilesFS, "dotfiles")
}