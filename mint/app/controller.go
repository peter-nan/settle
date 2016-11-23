package app

import (
	"github.com/spolu/settle/mint/endpoint"
	"goji.io"
	"goji.io/pat"
)

// Controller binds the API
type Controller struct{}

// Bind registers the API routes.
func (c *Controller) Bind(
	mux *goji.Mux,
) {
	// Local.
	mux.HandleFunc(pat.Post("/assets"), endpoint.HandlerFor(endpoint.EndPtCreateAsset))
	mux.HandleFunc(pat.Post("/assets/:asset/operations"), endpoint.HandlerFor(endpoint.EndPtCreateOperation))
	mux.HandleFunc(pat.Post("/offers"), endpoint.HandlerFor(endpoint.EndPtCreateOffer))

	// Mixed.
	mux.HandleFunc(pat.Post("/transactions"), endpoint.HandlerFor(endpoint.EndPtCreateTransaction))

	// Public.
	mux.HandleFunc(pat.Get("/offers/:offer"), endpoint.HandlerFor(endpoint.EndPtRetrieveOffer))
	mux.HandleFunc(pat.Get("/transactions/:transaction"), endpoint.HandlerFor(endpoint.EndPtRetrieveTransaction))

	//mux.HandleFunc(pat.Post("/propagations"), endpoint.HandlerFor(endpoint.EndPtCreatePropagation))
	//mux.HandleFunc(pat.Get("/operations/:operation"), endpoint.HandlerFor(endpoint.EndPtRetrieveOperation))

	//mux.HandleFunc(pat.Post("/transactions/:transaction/settle"), c.controller.SettleOperation)
}

// /operations > best-effort propagation
// /offers > best-effort propagation

// /transactions
// /transactions/:transaction/settle
