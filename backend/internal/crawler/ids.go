package crawler

import (
	"strings"

	"github.com/google/uuid"
)

func NewPageID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
