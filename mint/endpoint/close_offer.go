// OWNER: stan

package endpoint

import (
	"context"
	"net/http"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/async/task"
	"github.com/spolu/settle/mint/model"
	"goji.io/pat"
)

const (
	// EndPtCloseOffer creates a new offer.
	EndPtCloseOffer EndPtName = "CloseOffer"
)

func init() {
	registrar[EndPtCloseOffer] = NewCloseOffer
}

// CloseOffer closes an offer, making it unusable by transactions
type CloseOffer struct {
	ID    string
	Owner string
	Token string
}

// NewCloseOffer constructs and initialiezes the endpoint.
func NewCloseOffer(
	r *http.Request,
) (Endpoint, error) {
	return &CloseOffer{}, nil
}

// Validate validates the input parameters.
func (e *CloseOffer) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "offer"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Owner = *owner
	e.Token = *token

	return nil
}

// Execute executes the endpoint.
func (e *CloseOffer) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	offer, err := model.LoadCanonicalOfferByOwnerToken(ctx, e.Owner, e.Token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if offer == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "offer_not_found",
			"The offer you are trying to close does not exist: %s.",
			e.ID,
		))
	}

	offer.Status = mint.OfStClosed

	err = offer.Save(ctx)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	err = async.Queue(ctx, task.NewPropagateOffer(ctx, time.Now(), e.ID))
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"offer": format.JSONPtr(model.NewOfferResource(ctx, offer)),
	}, nil
}