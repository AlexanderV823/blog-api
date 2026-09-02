package repository

import "errors"

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrPostNotFound    = errors.New("post not found")
	ErrCommentNotFound = errors.New("comment not found")
)
