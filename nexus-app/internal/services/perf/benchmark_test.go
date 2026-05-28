package perf

import (
	"context"
	"testing"
)

func BenchmarkRunQuickProfile(b *testing.B) {
	for index := 0; index < b.N; index++ {
		if _, err := RunQuickProfile(context.Background(), b.TempDir(), Options{
			WorkspaceFiles: 40,
			ActivityLines:  120,
			DataRows:       160,
			ArtifactCount:  20,
			SearchResults:  25,
		}); err != nil {
			b.Fatal(err)
		}
	}
}
