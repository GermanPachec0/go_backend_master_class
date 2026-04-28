package tests_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"eats/backend/common/testutils"
)

func TestComponent_CriticalFlow(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)

	ctx := t.Context()

	country := testutils.GenerateRandomCountry()

	customerUUID := registerCustomerInCity(ctx, t, clients, country, "Some city")
	assert.NotEmpty(t, customerUUID)
}
