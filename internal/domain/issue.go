// Package domain holds the core types of the issue tracker.
// SQL も HTTP も HTML も知らない層。
package domain

import (
	"errors"
	"time"
	"unicode/utf8"
)

// Status is the state of an issue.
type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
)

// ParseStatus converts a query-string value into a Status.
// 空文字（= 絞り込みなし）は呼び出し側で扱うので、ここでは不正値として扱う。
func ParseStatus(s string) (Status, bool) {
	switch Status(s) {
	case StatusOpen:
		return StatusOpen, true
	case StatusClosed:
		return StatusClosed, true
	default:
		return "", false
	}
}

// Issue is one tracked issue.
type Issue struct {
	ID        int64
	Title     string
	Body      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IssueQuery describes a filtered, paged read of the issue list.
//
// service ではなく domain に置いてあるのは、store が service を import せずに
// リポジトリを実装できるようにするため（依存の向きを domain へ一方向に保つ）。
type IssueQuery struct {
	Q      string // タイトル・本文の部分一致。空なら絞り込まない
	Status Status // 空なら絞り込まない
	Limit  int
	Offset int
}

// ErrNotFound is returned when an issue does not exist.
var ErrNotFound = errors.New("issue not found")

// 入力エラー。handler はこれを見て 422 とフォームの再描画に振り分ける。
var (
	ErrTitleRequired = errors.New("title is required")
	ErrTitleTooLong  = errors.New("title is too long")
)

// MaxTitleLen is the maximum number of characters in a title.
const MaxTitleLen = 120

// ValidateTitle checks a title. 文字数は byte ではなく rune で数える。
func ValidateTitle(title string) error {
	switch {
	case title == "":
		return ErrTitleRequired
	case utf8.RuneCountInString(title) > MaxTitleLen:
		return ErrTitleTooLong
	default:
		return nil
	}
}
