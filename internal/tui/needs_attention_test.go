package tui

import (
	"testing"
	"time"

	"github.com/cheenu1092-oss/prflow/internal/gh"
)

func TestNeedsReReview(t *testing.T) {
	username := "reviewer"

	// Helper to format time
	formatTime := func(t time.Time) string {
		return t.Format(time.RFC3339)
	}

	baseTime := time.Now().Add(-48 * time.Hour)      // 2 days ago
	reviewTime := time.Now().Add(-24 * time.Hour)    // 1 day ago
	updateTimeOld := time.Now().Add(-25 * time.Hour) // Before review
	updateTimeNew := time.Now().Add(-1 * time.Hour)  // After review

	tests := []struct {
		name     string
		pr       *gh.PR
		username string
		want     bool
	}{
		{
			name:     "nil PR",
			pr:       nil,
			username: username,
			want:     false,
		},
		{
			name: "empty username",
			pr: &gh.PR{
				Number: 1,
				Author: gh.Author{Login: "author"},
			},
			username: "",
			want:     false,
		},
		{
			name: "my own PR",
			pr: &gh.PR{
				Number:    1,
				Author:    gh.Author{Login: username},
				UpdatedAt: formatTime(updateTimeNew),
			},
			username: username,
			want:     false,
		},
		{
			name: "never reviewed",
			pr: &gh.PR{
				Number:    1,
				Author:    gh.Author{Login: "author"},
				UpdatedAt: formatTime(updateTimeNew),
				Reviews: gh.Reviews{
					Nodes: []gh.Review{
						{
							Author:      gh.Author{Login: "other"},
							State:       "APPROVED",
							SubmittedAt: formatTime(reviewTime),
						},
					},
				},
			},
			username: username,
			want:     false,
		},
		{
			name: "reviewed but no updates since",
			pr: &gh.PR{
				Number:    1,
				Author:    gh.Author{Login: "author"},
				UpdatedAt: formatTime(updateTimeOld),
				Reviews: gh.Reviews{
					Nodes: []gh.Review{
						{
							Author:      gh.Author{Login: username},
							State:       "APPROVED",
							SubmittedAt: formatTime(reviewTime),
						},
					},
				},
			},
			username: username,
			want:     false,
		},
		{
			name: "reviewed and PR updated after my review",
			pr: &gh.PR{
				Number:    1,
				Author:    gh.Author{Login: "author"},
				UpdatedAt: formatTime(updateTimeNew),
				Reviews: gh.Reviews{
					Nodes: []gh.Review{
						{
							Author:      gh.Author{Login: username},
							State:       "CHANGES_REQUESTED",
							SubmittedAt: formatTime(reviewTime),
						},
					},
				},
			},
			username: username,
			want:     true,
		},
		{
			name: "multiple reviews, most recent is mine",
			pr: &gh.PR{
				Number:    1,
				Author:    gh.Author{Login: "author"},
				UpdatedAt: formatTime(updateTimeNew),
				Reviews: gh.Reviews{
					Nodes: []gh.Review{
						{
							Author:      gh.Author{Login: "other"},
							State:       "APPROVED",
							SubmittedAt: formatTime(baseTime),
						},
						{
							Author:      gh.Author{Login: username},
							State:       "CHANGES_REQUESTED",
							SubmittedAt: formatTime(reviewTime),
						},
					},
				},
			},
			username: username,
			want:     true,
		},
		{
			name: "case insensitive username match",
			pr: &gh.PR{
				Number:    1,
				Author:    gh.Author{Login: "Author"},
				UpdatedAt: formatTime(updateTimeNew),
				Reviews: gh.Reviews{
					Nodes: []gh.Review{
						{
							Author:      gh.Author{Login: "REVIEWER"},
							State:       "APPROVED",
							SubmittedAt: formatTime(reviewTime),
						},
					},
				},
			},
			username: "reviewer",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsReReview(tt.pr, tt.username)
			if got != tt.want {
				t.Errorf("needsReReview() = %v, want %v", got, tt.want)
			}
		})
	}
}
