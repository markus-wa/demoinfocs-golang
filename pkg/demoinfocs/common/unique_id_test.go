package common

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniqueID_String(t *testing.T) {
	assert.Equal(t, "0", UniqueID(0).String())
	assert.Equal(t, "42", UniqueID(42).String())
	assert.Equal(t, "18446744073709551615", UniqueID(1<<64-1).String())
	assert.Equal(t, "42", fmt.Sprint(UniqueID(42)))
}

func TestUniqueID_Uint64(t *testing.T) {
	assert.Equal(t, uint64(42), UniqueID(42).Uint64())
	assert.Equal(t, uint64(1<<64-1), UniqueID(1<<64-1).Uint64())
}

func TestUniqueID_JSON(t *testing.T) {
	type wrapper struct {
		ID UniqueID `json:"id"`
	}

	b, err := json.Marshal(wrapper{ID: 42})
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":42}`, string(b))

	var w wrapper

	require.NoError(t, json.Unmarshal(b, &w))
	assert.Equal(t, UniqueID(42), w.ID)
}
