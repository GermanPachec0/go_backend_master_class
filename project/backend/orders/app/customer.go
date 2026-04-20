package app

import (
	"eats/backend/common"
	"eats/backend/common/shared"
)

type Customer struct {
	UUID        common.UUID
	Name        string
	Email       string
	Address     shared.Address
	PhoneNumber string
}
