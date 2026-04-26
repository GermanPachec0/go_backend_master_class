package tests_test

import (
	"testing"

	"eats/backend/common/testutils"

	"github.com/stretchr/testify/assert"
)

func TestComponent_CriticalFlow(t *testing.T) {
	t.Parallel()
	country := testutils.GenerateRandomCountry()
	ctx := t.Context()
	customerUUID := registerCustomerInCity(ctx, t, newTestClients(t), country, "Some City")

	assert.NotEmpty(t, customerUUID.UUID.String())
}
