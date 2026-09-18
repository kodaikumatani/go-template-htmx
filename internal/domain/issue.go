// Package domain holds the core types of the issue tracker.
// SQL も HTTP も HTML も知らない層。
package domain

import (
	"errors"
	"time"
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
