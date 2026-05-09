package app

import (
	"context"
	"fmt"
	"strings"

	"eats/backend/common"
)

type CourierUUID struct {
	common.UUID
}

type RegisterCourier struct {
	Name        string
	PhoneNumber string
	City        string
}

type CourierRepository interface {
	RegisterCourier(ctx context.Context, courierUUID CourierUUID, courier RegisterCourier) error
}

func (s *Service) RegisterCourier(ctx context.Context, req RegisterCourier) (CourierUUID, error) {
	courierUUID := CourierUUID{common.NewUUIDv7()}

	var validationDetails []common.ErrorDetails

	if strings.TrimSpace(req.Name) == "" {
		validationDetails = append(validationDetails, common.ErrorDetails{
			EntityType: "courier",
			EntityID:   courierUUID.String(),
			ErrorSlug:  "invalid-name",
			Message:    "courier name cannot be empty",
		})
	}
	if strings.TrimSpace(req.PhoneNumber) == "" {
		validationDetails = append(validationDetails, common.ErrorDetails{
			EntityType: "courier",
			EntityID:   courierUUID.String(),
			ErrorSlug:  "invalid-phone-number",
			Message:    "courier phone number cannot be empty",
		})
	}
	if strings.TrimSpace(req.City) == "" {
		validationDetails = append(validationDetails, common.ErrorDetails{
			EntityType: "courier",
			EntityID:   courierUUID.String(),
			ErrorSlug:  "invalid-city",
			Message:    "courier city cannot be empty",
		})
	}
	if len(validationDetails) > 0 {
		return CourierUUID{}, common.NewInvalidInputError(
			"invalid-courier-data",
			"invalid courier data",
		).WithDetails(validationDetails)
	}

	err := s.courierRepository.RegisterCourier(ctx, courierUUID, req)
	if err != nil {
		return CourierUUID{}, err
	}

	return courierUUID, nil
}

func checkCustomerMatch(orderCustomer CustomerUUID, customerUUID CustomerUUID) error {
	if orderCustomer.Equals(customerUUID.UUID) {
		return nil
	}

	return common.NewForbiddenError(
		"invalid-customer",
		"order does not belong to the customer",
	).WithInternalError(fmt.Errorf(
		"order customer %s does not match provided customer %s",
		orderCustomer,
		customerUUID,
	))
}
