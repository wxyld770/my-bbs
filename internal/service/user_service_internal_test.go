package service

import (
	"testing"
	"time"
)

func TestInvitationRequiresPublishedPostBoundary(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		createdAt time.Time
		want      bool
	}{
		{name: "one millisecond before seven days", createdAt: now.Add(-InvitationNewUserPeriod + time.Millisecond), want: true},
		{name: "exactly seven days", createdAt: now.Add(-InvitationNewUserPeriod), want: false},
		{name: "older than seven days", createdAt: now.Add(-InvitationNewUserPeriod - time.Millisecond), want: false},
		{name: "legacy zero timestamp", createdAt: time.Time{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := invitationRequiresPublishedPost(tt.createdAt, now); got != tt.want {
				t.Fatalf("invitationRequiresPublishedPost()=%v, want %v", got, tt.want)
			}
		})
	}
}
