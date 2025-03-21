package stripe

import (
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/billingportal/session"
	"github.com/stripe/stripe-go/v81/customer"
)

func CreateCustomer(email string) (string, error) {
	// search for existing customer by email
	existing_customer := customer.List(
		&stripe.CustomerListParams{
			Email: stripe.String(email),
		},
	)
	if existing_customer.Err() != nil {
		return "", existing_customer.Err()
	}

	// check for existing customer
	if existing_customer.Next() {
		customer := existing_customer.Customer()
		return customer.ID, nil
	}

	// create new customer
	customer_params := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	customer, err := customer.New(customer_params)
	if err != nil {
		return "", err
	}

	return customer.ID, nil
}

func GetPortalSession(customerID string) (string, error) {
	session, err := session.New(&stripe.BillingPortalSessionParams{
		Customer: stripe.String(customerID),
	})
	if err != nil {
		return "", err
	}

	return session.URL, nil
}
