// Package service holds the use cases. HTTP も SQL も知らない。
package service

import (
	"context"
	"fmt"

	"github.com/kodaikumatani/go-template-htmx/internal/domain"
)

// IssueRepository is implemented by store.IssueStore.
//
// interface を利用側（service）で宣言するのが Go の作法。
// store 側は「service を知らないまま」この形を満たしている。
// Step が進むごとにここにメソッドが増えていく。
type IssueRepository interface {
	List(ctx context.Context, q domain.IssueQuery) (issues []domain.Issue, total int, err error)
}

// ListQuery is the input of the issue list use case.
// ページ番号は 1 始まり。ここが「URL の世界」との境界になる。
type ListQuery struct {
	Q       string
	Status  domain.Status
	Page    int
	PerPage int
}

// DefaultPerPage is used when ListQuery.PerPage is not set.
const DefaultPerPage = 5

const maxPerPage = 100

// normalize fills in defaults. 不正な値で SQL を組ませないための関所。
func (q ListQuery) normalize() ListQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PerPage < 1 {
		q.PerPage = DefaultPerPage
	}
	if q.PerPage > maxPerPage {
		q.PerPage = maxPerPage
	}
	return q
}

// ListResult is the output of the issue list use case.
type ListResult struct {
	Issues  []domain.Issue
	Total   int
	Page    int
	PerPage int
}

// TotalPages returns the number of pages, at least 1.
func (r ListResult) TotalPages() int {
	if r.Total == 0 {
		return 1
	}
	return (r.Total + r.PerPage - 1) / r.PerPage
}

func (r ListResult) HasPrev() bool { return r.Page > 1 }
func (r ListResult) HasNext() bool { return r.Page < r.TotalPages() }

// IssueService is the use case entry point for issues.
type IssueService struct {
	repo IssueRepository
}

func NewIssueService(repo IssueRepository) *IssueService {
	return &IssueService{repo: repo}
}

// List returns one page of issues.
func (s *IssueService) List(ctx context.Context, q ListQuery) (ListResult, error) {
	q = q.normalize()

	issues, total, err := s.repo.List(ctx, domain.IssueQuery{
		Q:      q.Q,
		Status: q.Status,
		Limit:  q.PerPage,
		Offset: (q.Page - 1) * q.PerPage,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list issues: %w", err)
	}

	return ListResult{
		Issues:  issues,
		Total:   total,
		Page:    q.Page,
		PerPage: q.PerPage,
	}, nil
}
