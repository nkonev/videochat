package handlers

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	shutdown()
	os.Exit(retCode)
}

func shutdown() {
}

func setup() {
}

func TestCode(t *testing.T) {
	policy := StripStripSourcePolicy()
	sanitized := policy.Sanitize(`<code>@admin</code><p>Hello @nikita</p>`)
	assert.Equal(t, "Hello @nikita", sanitized)
}

func TestPre(t *testing.T) {
	policy := StripStripSourcePolicy()
	sanitized := policy.Sanitize(`<pre>@admin</pre><p>Hello @nikita</p>`)
	assert.Equal(t, "Hello @nikita", sanitized)
}
