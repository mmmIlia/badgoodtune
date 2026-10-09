package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	ReviewHeadlineMaxLength = 120
	ReviewBodyMaxLength     = 10_000
	ReviewScoreMin          = 1
	ReviewScoreMax          = 10
)

type Review struct {
	ID        ID
	AlbumID   ID
	Author    User
	Headline  string
	Body      string
	Score     int
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReviewDraft struct {
	Headline string
	Body     string
	Score    int
}

type ReviewPatch struct {
	Headline *string
	Body     *string
	Score    *int
	Version  int64
}

func ValidateReviewDraft(draft ReviewDraft) (ReviewDraft, error) {
	draft.Headline = strings.TrimSpace(draft.Headline)
	draft.Body = strings.TrimSpace(draft.Body)

	violations := make([]Violation, 0, 3)
	violations = append(violations, validateText("headline", draft.Headline, ReviewHeadlineMaxLength)...)
	violations = append(violations, validateText("body", draft.Body, ReviewBodyMaxLength)...)
	if draft.Score < ReviewScoreMin || draft.Score > ReviewScoreMax {
		violations = append(violations, Violation{
			Field:       "score",
			Description: fmt.Sprintf("must be between %d and %d", ReviewScoreMin, ReviewScoreMax),
		})
	}

	return draft, NewValidationError(violations...)
}

func ApplyReviewPatch(review Review, patch ReviewPatch) (ReviewDraft, error) {
	violations := make([]Violation, 0, 2)
	if patch.Version < 1 {
		violations = append(violations, Violation{Field: "version", Description: "must be positive"})
	}
	if patch.Headline == nil && patch.Body == nil && patch.Score == nil {
		violations = append(violations, Violation{Field: "body", Description: "must change at least one field"})
	}
	if err := NewValidationError(violations...); err != nil {
		return ReviewDraft{}, err
	}

	draft := ReviewDraft{Headline: review.Headline, Body: review.Body, Score: review.Score}
	if patch.Headline != nil {
		draft.Headline = *patch.Headline
	}
	if patch.Body != nil {
		draft.Body = *patch.Body
	}
	if patch.Score != nil {
		draft.Score = *patch.Score
	}

	return ValidateReviewDraft(draft)
}
