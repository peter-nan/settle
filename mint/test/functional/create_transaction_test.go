package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupCreateTransaction(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser, []mint.AssetResource, []mint.OfferResource) {
	m := []*test.Mint{
		test.CreateMint(t),
		test.CreateMint(t),
		test.CreateMint(t),
	}
	u := []*test.MintUser{
		m[0].CreateUser(t),
		m[1].CreateUser(t),
		m[2].CreateUser(t),
	}
	a := []mint.AssetResource{
		u[0].CreateAsset(t, "USD", 2),
		u[1].CreateAsset(t, "USD", 2),
		u[2].CreateAsset(t, "USD", 2),
	}

	o := []mint.OfferResource{
		u[0].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address),
			"100/100", big.NewInt(100)),
		u[1].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[1].Address, u[0].Address),
			"100/100", big.NewInt(100)),
		u[2].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[2].Address, u[1].Address),
			"100/98", big.NewInt(100)),
	}

	return m, u, a, o
}

func tearDownCreateTransaction(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestCreateTransactionWith2Offers(
	t *testing.T,
) {
	t.Parallel()
	m, u, _, o := setupCreateTransaction(t)
	defer tearDownCreateTransaction(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]": {
				o[1].ID,
				o[2].ID,
			},
		})

	var tx0 mint.TransactionResource
	if err := raw.Extract("transaction", &tx0); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, tx0.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx0.Created*mint.TimeResolutionNs), 10*test.PostLatency)
	assert.Equal(t, u[0].Address, tx0.Owner)

	assert.Equal(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address),
		tx0.Pair)
	assert.Equal(t, big.NewInt(10), tx0.Amount)
	assert.Equal(t, u[2].Address, tx0.Destination)
	assert.Equal(t, []string{o[1].ID, o[2].ID}, tx0.Path)
	assert.Equal(t, mint.TxStReserved, tx0.Status)
	assert.Equal(t, 1, len(tx0.Operations))
	assert.Equal(t, []mint.CrossingResource{}, tx0.Crossings)

	assert.Regexp(t, mint.IDRegexp, tx0.Operations[0].ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx0.Operations[0].Created*mint.TimeResolutionNs),
		10*test.PostLatency)
	assert.Equal(t, u[0].Address, tx0.Operations[0].Owner)
	assert.Equal(t,
		fmt.Sprintf("%s[USD.2]", u[0].Address), tx0.Operations[0].Asset)
	assert.Equal(t, u[0].Address, tx0.Operations[0].Source)
	assert.Equal(t, u[1].Address, tx0.Operations[0].Destination)
	assert.Equal(t, big.NewInt(11), tx0.Operations[0].Amount)
	assert.Equal(t, mint.TxStReserved, tx0.Operations[0].Status)
	assert.Equal(t, tx0.ID, *tx0.Operations[0].Transaction)
	assert.Equal(t, int8(0), *tx0.Operations[0].TransactionHop)

	// Check transaction on m[1].
	status, raw = u[1].Get(t, fmt.Sprintf("/transactions/%s", tx0.ID))

	var tx1 mint.TransactionResource
	if err := raw.Extract("transaction", &tx1); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, status)
	assert.Equal(t, tx0.ID, tx1.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx1.Created*mint.TimeResolutionNs), 10*test.PostLatency)
	assert.Equal(t, u[0].Address, tx1.Owner)

	assert.Equal(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address),
		tx1.Pair)
	assert.Equal(t, big.NewInt(10), tx1.Amount)
	assert.Equal(t, u[2].Address, tx1.Destination)
	assert.Equal(t, []string{o[1].ID, o[2].ID}, tx1.Path)
	assert.Equal(t, mint.TxStReserved, tx1.Status)
	assert.Equal(t, tx0.Lock, tx1.Lock)
	assert.Equal(t, 1, len(tx1.Operations))
	assert.Equal(t, 1, len(tx1.Crossings))

	assert.Regexp(t, mint.IDRegexp, tx1.Crossings[0].ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx1.Crossings[0].Created*mint.TimeResolutionNs),
		10*test.PostLatency)
	assert.Equal(t, u[1].Address, tx1.Crossings[0].Owner)
	assert.Equal(t, o[1].ID, tx1.Crossings[0].Offer)
	assert.Equal(t, big.NewInt(11), tx1.Crossings[0].Amount)
	assert.Equal(t, mint.TxStReserved, tx1.Crossings[0].Status)
	assert.Equal(t, tx1.ID, tx1.Crossings[0].Transaction)
	assert.Equal(t, int8(1), tx1.Crossings[0].TransactionHop)

	assert.Regexp(t, mint.IDRegexp, tx1.Operations[0].ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx1.Operations[0].Created*mint.TimeResolutionNs),
		10*test.PostLatency)
	assert.Equal(t, u[1].Address, tx1.Operations[0].Owner)
	assert.Equal(t,
		fmt.Sprintf("%s[USD.2]", u[1].Address), tx1.Operations[0].Asset)
	assert.Equal(t, u[1].Address, tx1.Operations[0].Source)
	assert.Equal(t, u[2].Address, tx1.Operations[0].Destination)
	assert.Equal(t, big.NewInt(11), tx1.Operations[0].Amount)
	assert.Equal(t, mint.TxStReserved, tx1.Operations[0].Status)
	assert.Equal(t, tx1.ID, *tx1.Operations[0].Transaction)
	assert.Equal(t, int8(2), *tx1.Operations[0].TransactionHop)

	// Check transaction on m[2].
	status, raw = u[2].Get(t, fmt.Sprintf("/transactions/%s", tx0.ID))

	var tx2 mint.TransactionResource
	if err := raw.Extract("transaction", &tx2); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, status)
	assert.Equal(t, tx0.ID, tx2.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx2.Created*mint.TimeResolutionNs), 10*test.PostLatency)
	assert.Equal(t, u[0].Address, tx2.Owner)

	assert.Equal(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address),
		tx2.Pair)
	assert.Equal(t, big.NewInt(10), tx2.Amount)
	assert.Equal(t, u[2].Address, tx2.Destination)
	assert.Equal(t, []string{o[1].ID, o[2].ID}, tx2.Path)
	assert.Equal(t, mint.TxStReserved, tx2.Status)
	assert.Equal(t, tx0.Lock, tx2.Lock)
	assert.Equal(t, 1, len(tx2.Operations))
	assert.Equal(t, 1, len(tx2.Crossings))

	assert.Regexp(t, mint.IDRegexp, tx2.Crossings[0].ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx2.Crossings[0].Created*mint.TimeResolutionNs),
		10*test.PostLatency)
	assert.Equal(t, u[2].Address, tx2.Crossings[0].Owner)
	assert.Equal(t, o[2].ID, tx2.Crossings[0].Offer)
	assert.Equal(t, big.NewInt(11), tx2.Crossings[0].Amount)
	assert.Equal(t, mint.TxStReserved, tx2.Crossings[0].Status)
	assert.Equal(t, tx2.ID, tx2.Crossings[0].Transaction)
	assert.Equal(t, int8(3), tx2.Crossings[0].TransactionHop)

	assert.Regexp(t, mint.IDRegexp, tx2.Operations[0].ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, tx2.Operations[0].Created*mint.TimeResolutionNs),
		10*test.PostLatency)
	assert.Equal(t, u[2].Address, tx2.Operations[0].Owner)
	assert.Equal(t,
		fmt.Sprintf("%s[USD.2]", u[2].Address), tx2.Operations[0].Asset)
	assert.Equal(t, u[2].Address, tx2.Operations[0].Source)
	assert.Equal(t, u[2].Address, tx2.Operations[0].Destination)
	assert.Equal(t, big.NewInt(10), tx2.Operations[0].Amount)
	assert.Equal(t, mint.TxStReserved, tx2.Operations[0].Status)
	assert.Equal(t, tx2.ID, *tx2.Operations[0].Transaction)
	assert.Equal(t, int8(4), *tx2.Operations[0].TransactionHop)
}

func TestCreateTransactionWithInsufficientOfferAmount(
	t *testing.T,
) {
	t.Parallel()
	m, u, _, o := setupCreateTransaction(t)
	defer tearDownCreateTransaction(t, m)

	o1 := u[1].CreateOffer(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[1].Address, u[0].Address),
		"100/100", big.NewInt(5))

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]": {
				o1.ID,
				o[2].ID,
			},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 402, status)
	assert.Equal(t, "transaction_failed", e.ErrCode)
}

func TestCreateTransactionWithUserUsedTwice(
	t *testing.T,
) {
	t.Parallel()
	m, u, _, _ := setupCreateTransaction(t)
	defer tearDownCreateTransaction(t, m)

	m3 := test.CreateMint(t)
	defer m3.Close()

	u3 := m3.CreateUser(t)
	u3.CreateAsset(t, "USD", 2)

	// Create an offer chain that uses a user twice with a positive loop.
	o1 := u[1].CreateOffer(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[1].Address, u[0].Address),
		"100/100", big.NewInt(100))
	o2 := u3.CreateOffer(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u3.Address, u[1].Address),
		"100/120", big.NewInt(100))
	o3 := u[1].CreateOffer(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[1].Address, u3.Address),
		"100/100", big.NewInt(100))
	o4 := u[2].CreateOffer(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[2].Address, u[1].Address),
		"100/98", big.NewInt(100))

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]": {
				o1.ID, o2.ID, o3.ID, o4.ID,
			},
		})

	var tx0 mint.TransactionResource
	if err := raw.Extract("transaction", &tx0); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Equal(t, big.NewInt(10), tx0.Operations[0].Amount)
}

func TestCreateTransactionWithNoOffer(
	t *testing.T,
) {
	t.Parallel()
	m, u, _, _ := setupCreateTransaction(t)
	defer tearDownCreateTransaction(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[0].Address)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]":      {},
		})

	var tx0 mint.TransactionResource
	if err := raw.Extract("transaction", &tx0); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, tx0.ID)
	assert.Equal(t, big.NewInt(10), tx0.Operations[0].Amount)
	assert.Equal(t, u[2].Address, tx0.Operations[0].Destination)
	assert.Equal(t, u[0].Address, tx0.Operations[0].Source)
}

func TestCreateTransactionWith1Offer(
	t *testing.T,
) {
	t.Parallel()
	m, u, _, o := setupCreateTransaction(t)
	defer tearDownCreateTransaction(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[1].Address)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]": {
				o[1].ID,
			},
		})

	var tx0 mint.TransactionResource
	if err := raw.Extract("transaction", &tx0); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, tx0.ID)
	assert.Equal(t, big.NewInt(10), tx0.Operations[0].Amount)
	assert.Equal(t, u[1].Address, tx0.Operations[0].Destination)
	assert.Equal(t, u[0].Address, tx0.Operations[0].Source)

	// Check transaction on m[1].
	status, raw = u[1].Get(t, fmt.Sprintf("/transactions/%s", tx0.ID))

	var tx1 mint.TransactionResource
	if err := raw.Extract("transaction", &tx1); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, status)
	assert.Equal(t, big.NewInt(10), tx1.Crossings[0].Amount)
	assert.Equal(t, mint.TxStReserved, tx1.Crossings[0].Status)
	assert.Equal(t, tx1.ID, tx1.Crossings[0].Transaction)
	assert.Equal(t, int8(1), tx1.Crossings[0].TransactionHop)

	assert.Equal(t, u[1].Address, tx1.Operations[0].Source)
	assert.Equal(t, u[2].Address, tx1.Operations[0].Destination)
	assert.Equal(t, big.NewInt(10), tx1.Operations[0].Amount)
	assert.Equal(t, mint.TxStReserved, tx1.Operations[0].Status)
	assert.Equal(t, tx1.ID, *tx1.Operations[0].Transaction)
	assert.Equal(t, int8(2), *tx1.Operations[0].TransactionHop)

}
