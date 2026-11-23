// Package repository contains all MongoDB access. Repositories know nothing about HTTP or permissions.
package repository

import "errors"

var (
	ErrNotFound  = errors.New("document not found")
	ErrDuplicate = errors.New("duplicate key")
)
