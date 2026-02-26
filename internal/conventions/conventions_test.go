package conventions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderToURLPattern(t *testing.T) {
	tests := []struct {
		folder string
		want   string
	}{
		{"index", "/"},
		{"dashboard", "/dashboard"},
		{"about", "/about"},
		{"users.$id", "/users/{id}"},
		{"users.$id.edit", "/users/{id}/edit"},
		{"posts.$slug", "/posts/{slug}"},
		{"settings.billing", "/settings/billing"},
		{"org.$orgId.members.$memberId", "/org/{orgId}/members/{memberId}"},
	}
	for _, tt := range tests {
		got := FolderToURLPattern(tt.folder)
		assert.Equal(t, tt.want, got, "FolderToURLPattern(%q)", tt.folder)
	}
}

func TestIsRouteDir(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"routes", true},
		{"routes/dashboard", true},
		{"routes/users.$id.edit", true},
		{"shared/ui/button", false},
		{"shared/hooks", false},
		{"main", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsRouteDir(tt.path)
		assert.Equal(t, tt.want, got, "IsRouteDir(%q)", tt.path)
	}
}
