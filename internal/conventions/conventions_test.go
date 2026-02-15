package conventions

import "testing"

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
		if got != tt.want {
			t.Errorf("FolderToURLPattern(%q) = %q, want %q", tt.folder, got, tt.want)
		}
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
		if got != tt.want {
			t.Errorf("IsRouteDir(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
