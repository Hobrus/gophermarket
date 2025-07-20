package domain

import "errors"

var (
	// ErrConflictSelf indicates the order already belongs to the same user.
	ErrConflictSelf = errors.New("conflict: already uploaded by user")
	// ErrConflictOther indicates the order already belongs to another user.
	ErrConflictOther = errors.New("conflict: already uploaded by other user")
	// ErrNotFound is returned when requested entity is not found.
	ErrNotFound = errors.New("not found")
	// ErrInsufficientFunds indicates not enough balance for withdrawal.
	ErrInsufficientFunds = errors.New("insufficient funds")
)
