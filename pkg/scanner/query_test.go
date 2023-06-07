package scanner

import (
	"context"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	os.Exit(m.Run())
}

func TestOSVQuery(t *testing.T) {
	commit := "6879efc2c1596d11a6a6ad296f80063b558d5e0f"
	ctx := context.Background()

	want := &RequestResult{
		Vulnerabilities: []Vulnerability{
			{
				ID: "OSV-2020-484",
				Affected: []Affected{
					{
						Data: map[string]interface{}{
							"severity": "MEDIUM",
						},
					},
				},
			},
		},
	}

	got, err := Query(ctx, commit)
	assert.NoError(t, err, "Query() error = %v", err)
	assert.Equal(t, want, got, "Query() got = %v, want %v", got, want)
}
