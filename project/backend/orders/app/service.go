package app

type ModulesContract interface{}

type Service struct {
	customerRepository   CustomerRepository
	restaurantRepository RestaurantRepository
	modules              ModulesContract
}

func NewService(
	customerRepository CustomerRepository,
	restaurantRepository RestaurantRepository,
	modules ModulesContract,
) *Service {
	if customerRepository == nil {
		panic("customerRepository cannot be nil")
	}

	if restaurantRepository == nil {
		panic("restaurantRepository cannot be nil")
	}

	if modules == nil {
		panic("modules cannot be nil")
	}

	return &Service{
		customerRepository:   customerRepository,
		restaurantRepository: restaurantRepository,
		modules:              modules,
	}
}
