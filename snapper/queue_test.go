package snapper_test

import (
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/snapper-producer-go/snapper"
)

func TestQueue(t *testing.T) {

	t.Run("Queue that should be", func(t *testing.T) {
		assert := assert.New(t)
		queue := snapper.NewQueue()
		for w := 0; w < 1000; w++ {
			queue.Push([]string{strconv.Itoa(w)})
		}
		arr := queue.PopN(10)
		assert.Len(arr, 10)
		assert.Equal([]string{"9"}, arr[9])
		assert.Equal([]string{"0"}, arr[0])
		a := queue.Pop()
		assert.Equal([]string{"10"}, a)

		a = queue.Pop()
		assert.Equal([]string{"11"}, a)

		a = queue.Pop()
		assert.Equal([]string{"12"}, a)

		queue = snapper.NewQueue()
		queue.Push([]string{""})
		b := queue.PopN(2)
		assert.Equal(1, len(b))

	})
}
