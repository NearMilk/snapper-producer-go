package snapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnector(t *testing.T) {
	t.Run("Connector with error message that should be", func(t *testing.T) {

		assert := assert.New(t)

		conn := newConnector("127.0.0.1:7720", nil)

		err := conn.validateAuth("xx")
		assert.Contains(err.Error(), "Parse error")

	})
}
