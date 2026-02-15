package renderer

import (
	"github.com/sgnl-ai/cclog/internal/parser"
)

// Options configures the rendering output.
type Options struct {
	Messages []parser.Message
	Title    string
}
