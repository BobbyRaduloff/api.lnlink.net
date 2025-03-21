package stripe

import (
	"api.lnlink.net/src/pkg/global"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

func CreateCheckoutSession(customerID string, priceID string) (string, error) {
	session, err := session.New(&stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		Mode:       stripe.String("payment"),
		SuccessURL: stripe.String(global.SUCCESS_URL),
		CancelURL:  stripe.String(global.SUCCESS_URL),
		TaxIDCollection: &stripe.CheckoutSessionTaxIDCollectionParams{
			Enabled: stripe.Bool(true),
		},
		CustomerUpdate: &stripe.CheckoutSessionCustomerUpdateParams{
			Address: stripe.String("auto"),
			Name:    stripe.String("auto"),
		},
	})
	if err != nil {
		return "", err
	}

	return session.URL, nil
}
